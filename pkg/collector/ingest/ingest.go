package ingest

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
	"github.com/google/pprof/profile"
	"github.com/rancher-sandbox/profiling/pkg/collector/labels"
	"github.com/rancher-sandbox/profiling/pkg/collector/storage"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	colprofilespb "go.opentelemetry.io/proto/otlp/collector/profiles/v1development"
	profilespb "go.opentelemetry.io/proto/otlp/profiles/v1development"
)

const (
	pbContentType   = "application/x-protobuf"
	jsonContentType = "application/json"
)

type OTLPIngester struct {
	logger *slog.Logger
	store  storage.Store

	colprofilespb.UnsafeProfilesServiceServer
}

func NewOTLPIngester(logger *slog.Logger, store storage.Store) *OTLPIngester {
	return &OTLPIngester{
		logger: logger,
		store:  store,
	}
}

func (o *OTLPIngester) StartHTTP(addr string) error {
	logger := o.logger.With("http-addr", addr)
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	logger.Info("Configuring otlphttp routers")
	o.ConfigureRoutes(router)

	go func() {
		logger.Info("Starting otlphttp ingestion server...")
		if err := router.Run(addr); err != nil {
			logger.With("error", err).Error("failed to run router")
			return
		}
	}()

	return nil
}

func (o *OTLPIngester) StartGrpc(addr string) error {
	url, err := url.Parse(addr)
	if err != nil {
		o.logger.With("error", err).Error("failed to parse address")
		return err
	}

	o.logger.With("grpc-addr", addr).Info("Starting OTLP ingestion server")
	listener, err := net.Listen(url.Scheme, url.Host)
	if err != nil {
		o.logger.With("error", err).Error("failed to listen")
		return err
	}

	server := grpc.NewServer(
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             15 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    15 * time.Second,
			Timeout: 5 * time.Second,
		}),
	)

	server.RegisterService(&colprofilespb.ProfilesService_ServiceDesc, o)
	go func() {
		if err := server.Serve(listener); err != nil {
			o.logger.With("error", err).Error("failed to serve")
			return
		}
	}()
	return nil
}

var _ colprofilespb.ProfilesServiceServer = (*OTLPIngester)(nil)

func (o *OTLPIngester) ConfigureRoutes(router *gin.Engine) {
	router.POST("/v1/development/profiles", o.handleProfilesPost)
}

func (o *OTLPIngester) handleProfilesPost(c *gin.Context) {
	switch c.ContentType() {
	case pbContentType:
		o.renderProto(c)
	case jsonContentType:
		o.renderProtoJSON(c)
	default:
		c.String(http.StatusUnsupportedMediaType, "unsupported media type, supported: [%s,%s]", jsonContentType, pbContentType)
		return
	}
}

func (o *OTLPIngester) Export(_ context.Context, req *colprofilespb.ExportProfilesServiceRequest) (*colprofilespb.ExportProfilesServiceResponse, error) {
	// TODO : eventually identify different sources, now we assume these are coming from eBPF collector
	partialResp := o.handleEbpfCollectorProfile(req.GetResourceProfiles())
	return &colprofilespb.ExportProfilesServiceResponse{
		PartialSuccess: partialResp,
	}, nil
}

func splitByPid(profile *profilespb.Profile) map[int64][]*profilespb.Profile {
	ret := map[int64][]*profilespb.Profile{}
	for idx, s := range profile.GetSample() {
		var pid *int64
		for _, attrIdx := range s.GetAttributeIndices() {
			attr := profile.GetAttributeTable()[attrIdx]
			if attr.GetKey() == "process.pid" {
				pid = lo.ToPtr(attr.GetValue().GetIntValue())
			}
		}
		if pid == nil {
			logrus.Warnf("dropping sample for sample at index : %d", idx)
			continue
		}
		// hack: copy the profile
		data, err := proto.Marshal(profile)
		if err != nil {
			panic(err)
		}
		var base profilespb.Profile
		if err := proto.Unmarshal(data, &base); err != nil {
			panic(err)
		}

		base.Sample = []*profilespb.Sample{s}
		if _, ok := ret[*pid]; !ok {
			ret[*pid] = []*profilespb.Profile{&base}
		} else {
			ret[*pid] = append(ret[*pid], &base)
		}
	}
	return ret

}

func (o *OTLPIngester) handleEbpfCollectorProfile(rscs []*profilespb.ResourceProfiles) *colprofilespb.ExportProfilesPartialSuccess {
	failedCount := int64(0)
	errs := []error{}
	for _, rsc := range rscs {
		// resource from opentelemetry-eBPF-profiler has attributes
		// 	- host.id
		//  - host.ip
		// 	- host.name
		//  - service.version
		//  - os.kernel

		for _, scope := range rsc.GetScopeProfiles() {
			for _, prof := range scope.GetProfiles() {
				// split by PID
				profileMap := splitByPid(prof)
				for pid, profiles := range profileMap {
					threadNames := []string{}
					// logrus.Infof("split result (pid=%d) : num profiles : %d", pid, len(profiles))
					var baseProfile *profile.Profile
					for _, prof := range profiles {
						for _, s := range prof.GetSample() {
							for _, attrIdx := range s.GetAttributeIndices() {
								attr := prof.GetAttributeTable()[attrIdx]
								if attr.GetKey() == "thread.name" {
									name := attr.GetValue().GetStringValue()
									name = strings.ReplaceAll(name, "/", "-")
									threadNames = append(threadNames, name)
									if name == "collector" {
										logrus.Debug("collector thread")
									}
								}
							}
						}

						p := Convert(prof)
						if p == nil {
							panic("failed to convert to pprof profile")
						}
						if err := p.CheckValid(); err != nil {
							panic(err)
						}
						if baseProfile == nil {
							baseProfile = p
						} else {
							pf, err := profile.Merge([]*profile.Profile{baseProfile, p})
							if err != nil {
								panic(err)
							}
							baseProfile = pf
						}
					}

					b := bytes.NewBuffer([]byte{})
					if err := baseProfile.Write(b); err != nil {
						panic(err)
					}
					threadSuffix := strings.Join(lo.Uniq(threadNames), "-")
					if err := o.store.Put(time.Now(), time.Now(), "profile", fmt.Sprintf("pid-%d-%s", pid, threadSuffix), map[string]string{
						labels.NamespaceLabel: "ebpf-local",
						labels.NameLabel:      "host",
					}, b.Bytes()); err != nil {
						panic(err)
					}
				}

				p := Convert(prof)
				if err := p.CheckValid(); err != nil {
					failedCount += 1
					o.logger.With("error", err).Error("cannot convert to pprof profile")
					errs = append(errs, fmt.Errorf("failed to convert to pprof profile : %w", err))
					continue
				}
				b := bytes.NewBuffer([]byte{})
				if err := p.Write(b); err != nil {
					failedCount += 1
					o.logger.With("error", err).Error("failed to write profile to buffer")
					errs = append(errs, fmt.Errorf("failed to write profile to buffer: %w", err))
					continue
				}
				const allKey = "all"
				if err := o.store.Put(time.Now(), time.Now(), "profile", allKey, map[string]string{
					labels.NamespaceLabel: "ebpf-local",
					labels.NameLabel:      "host",
				}, b.Bytes()); err != nil {
					failedCount += 1
					o.logger.With("error", err).Error("failed to store profile")
					errs = append(errs, fmt.Errorf("failed to store profile: %w", err))
					continue
				}
			}
		}
	}
	err := errors.Join(errs...)
	var msg string
	if err != nil {
		msg = err.Error()
	}
	return &colprofilespb.ExportProfilesPartialSuccess{
		RejectedProfiles: failedCount,
		ErrorMessage:     msg,
	}
}

func uniqueFunctionIDx(strTableLen int, fnIdx, systemIdx int32) uint64 {
	var factor int
	if strTableLen <= 10 {
		factor = 10
	} else {
		factor = int(math.Pow(10, math.Ceil(math.Log10(float64(strTableLen)))))
	}
	return uint64(fnIdx+int32(factor)*int32(systemIdx)) + 1
}

func isEmptyFunction(f *profilespb.Function) bool {
	maybeEmpty := f.NameStrindex == 0 && f.FilenameStrindex == 0 && f.SystemNameStrindex == 0
	if maybeEmpty {
		logrus.Debug("maybe empty function")
	}
	return maybeEmpty
}

func Convert(p *profilespb.Profile) *profile.Profile {

	// end debug
	out := &profile.Profile{
		SampleType: []*profile.ValueType{},
		Sample:     []*profile.Sample{},
		Mapping:    []*profile.Mapping{},
		Location:   []*profile.Location{},
		Function:   []*profile.Function{},
	}

	for _, st := range p.GetSampleType() {
		out.SampleType = append(out.SampleType, &profile.ValueType{
			Type: p.GetStringTable()[st.GetTypeStrindex()],
			Unit: p.GetStringTable()[st.GetUnitStrindex()],
		})
	}

	for _, f := range p.GetFunctionTable() {
		// TODO : check the spec, is the first function always supposed to be empty?
		if isEmptyFunction(f) {
			continue
		}

		appF := &profile.Function{
			ID:         uniqueFunctionIDx(len(p.GetStringTable()), f.GetNameStrindex(), f.GetSystemNameStrindex()),
			Name:       p.GetStringTable()[f.GetNameStrindex()],
			Filename:   p.GetStringTable()[f.GetFilenameStrindex()],
			SystemName: p.GetStringTable()[f.GetSystemNameStrindex()],
			StartLine:  f.GetStartLine(),
		}
		if appF.SystemName != "" {
			logrus.Info(appF.SystemName)
		}
		if appF.Filename != "" {
			logrus.Warn(appF.Filename, "|", appF.Name, "|", appF.SystemName)
		}

		if appF.Name == "" && appF.SystemName == "" && appF.Filename != "" {
			logrus.Warnf("must symbolize function with : %s", appF.Filename)
		}

		out.Function = append(out.Function, appF)

	}

	for _, f := range out.Function {
		if f.Name == "" && f.Filename == "" && f.SystemName == "" {
			panic("invalid function")
		}
	}

	for mIdx, m := range p.GetMappingTable() {
		out.Mapping = append(out.Mapping, &profile.Mapping{
			// FIXME:
			ID:              uint64(mIdx + 1),
			Start:           m.GetMemoryStart(),
			Limit:           m.GetMemoryLimit(),
			Offset:          m.GetFileOffset(),
			File:            p.GetStringTable()[m.GetFilenameStrindex()],
			HasFunctions:    m.GetHasFunctions(),
			HasFilenames:    m.GetHasFilenames(),
			HasLineNumbers:  m.GetHasLineNumbers(),
			HasInlineFrames: m.GetHasInlineFrames(),
		})
	}

	for lIdx, l := range p.GetLocationTable() {
		nl := &profile.Location{
			//FIXME:
			ID:       uint64(lIdx + 1),
			Address:  l.GetAddress(),
			IsFolded: l.GetIsFolded(),
			Line:     []profile.Line{},
			Mapping:  nil,
		}

		//FIXME:
		mapIdx := l.GetMappingIndex()
		if mapIdx >= 0 {
			nl.Mapping = out.Mapping[mapIdx]
		}

		for _, line := range l.GetLine() {
			fn := p.GetFunctionTable()[line.FunctionIndex]
			if isEmptyFunction(fn) {
				continue
			}
			nl.Line = append(nl.Line, profile.Line{
				Function: &profile.Function{
					//FIXME:
					ID:         uniqueFunctionIDx(len(p.GetStringTable()), fn.GetNameStrindex(), fn.GetSystemNameStrindex()),
					Name:       p.GetStringTable()[fn.GetNameStrindex()],
					Filename:   p.GetStringTable()[fn.GetFilenameStrindex()],
					SystemName: p.GetStringTable()[fn.GetSystemNameStrindex()],
					StartLine:  fn.GetStartLine(),
				},
				Line:   line.GetLine(),
				Column: line.GetLine(),
			})
		}
		out.Location = append(out.Location, nl)
	}

	for idx, s := range p.GetSample() {
		var threadName string
		var pid int64
		for _, attrIdx := range s.GetAttributeIndices() {
			attr := p.GetAttributeTable()[attrIdx]
			logrus.Debug(attr.GetKey())
			if attr.GetKey() == "thread.name" {
				threadName = attr.Value.GetStringValue()
			}
			if attr.GetKey() == "process.pid" {
				pid = attr.Value.GetIntValue()
			}
		}
		logrus.Debugf("[%d] pid: %d, thread.name: %s", idx, pid, threadName)

		// here we want to group by (pid thread.name) maybe?

		ps := &profile.Sample{
			Value:    s.GetValue(),
			Location: []*profile.Location{},
			Label:    map[string][]string{},
			NumLabel: map[string][]int64{},
			NumUnit:  map[string][]string{},
		}

		// first location to group by process/thread, purely "cosmetic"

		// ps.Location = append(ps.Location, &profile.Location{
		// 	ID: 7777 + uint64(idx),
		// 	Line: []profile.Line{
		// 		{
		// 			Function: &profile.Function{
		// 				ID:   7777 + uint64(idx),
		// 				Name: threadName,
		// 			},
		// 		},
		// 	},
		// 	Mapping:  &profile.Mapping{},
		// 	IsFolded: false,
		// })

		lStart := s.GetLocationsStartIndex()
		lEnd := s.GetLocationsLength()

		lIdxs := p.GetLocationIndices()[lStart : lStart+lEnd]

		for _, lIdx := range lIdxs {
			loc := p.GetLocationTable()[lIdx]
			nl := &profile.Location{
				//FIXME:
				ID:       uint64(lIdx) + 1,
				Address:  loc.GetAddress(),
				IsFolded: loc.GetIsFolded(),
				Line:     []profile.Line{},
				Mapping:  nil,
			}

			//FIXME:
			mapIdx := loc.GetMappingIndex()
			if mapIdx >= 0 {
				nl.Mapping = out.Mapping[mapIdx]
			}

			for _, line := range loc.GetLine() {
				fn := p.GetFunctionTable()[line.FunctionIndex]
				if isEmptyFunction(fn) {
					continue
				}
				nl.Line = append(nl.Line, profile.Line{
					Function: &profile.Function{
						//FIXME:
						ID:        uniqueFunctionIDx(len(p.GetStringTable()), fn.GetNameStrindex(), fn.GetSystemNameStrindex()),
						Name:      p.GetStringTable()[fn.GetNameStrindex()],
						Filename:  p.GetStringTable()[fn.GetFilenameStrindex()],
						StartLine: fn.GetStartLine(),
					},
					Line:   line.GetLine(),
					Column: line.GetLine(),
				})
			}
			ps.Location = append(ps.Location, nl)
		}
		out.Sample = append(out.Sample, ps)
	}
	// FIXME: hack, `*profile.Profile.CheckValid() checks pointer equality, when *profile.Parse is called`
	// it correctly unmarshals the pointers, the way we use of creating new functions above doesn't pass the pointer equality check
	b := bytes.NewBuffer([]byte{})
	if err := out.Write(b); err != nil {
		panic(err)
	}

	constructed, err := profile.Parse(b)
	if err != nil {
		panic(err)
	}
	return constructed
}

func (o *OTLPIngester) renderProto(c *gin.Context) {
	body, err := readBody(c)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	req := &colprofilespb.ExportProfilesServiceRequest{}
	err = proto.Unmarshal(body, req)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	otlpResp, err := o.Export(c.Request.Context(), req)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Render(http.StatusOK, render.ProtoBuf{
		Data: otlpResp,
	})
}

func (o *OTLPIngester) renderProtoJSON(c *gin.Context) {
	body, err := readBody(c)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	req := &colprofilespb.ExportProfilesServiceRequest{}
	err = protojson.Unmarshal(body, req)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	otlpResp, err := o.Export(c.Request.Context(), req)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Render(http.StatusOK, protoJSON{
		Data: otlpResp,
	})
}

func readBody(c *gin.Context) ([]byte, error) {
	bodyReader := c.Request.Body
	if c.GetHeader("Content-Encoding") == "gzip" {
		gr, err := gzip.NewReader(c.Request.Body)
		if err != nil {
			return []byte{}, err
		}
		defer gr.Close()
		bodyReader = gr
	}
	return io.ReadAll(bodyReader)
}

type protoJSON struct {
	Data protoreflect.ProtoMessage
}

func (p protoJSON) Render(w http.ResponseWriter) error {
	p.WriteContentType(w)

	bytes, err := protojson.Marshal(p.Data)
	if err != nil {
		return err
	}

	_, err = w.Write(bytes)
	return err
}

func (p protoJSON) WriteContentType(w http.ResponseWriter) {
	header := w.Header()
	if val := header["Content-Type"]; len(val) == 0 {
		header["Content-Type"] = []string{jsonContentType}
	}
}
