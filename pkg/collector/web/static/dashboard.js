let isHoveringIframe = false;
document.addEventListener("DOMContentLoaded", () => {
    console.log("DOM fully loaded");
    //document.body.addEventListener("wheel", onWheel);
    document.body.addEventListener('mousewheel DOMMouseScroll', onWheel);
    function onWheel (e){
        console.log(e.target);
        if (isHoveringIframe)
            e.preventDefault();
        console.log(e);
    }
    
    document.querySelectorAll("iframe").forEach(iframe => {
        iframe.onload = function() {
            const iframeDocument = iframe.contentWindow.document;

            // Intercept wheel events in the iframe's content
            iframeDocument.addEventListener('wheel', event => {
                    console.log(isHoveringIframe);
                    if (isHoveringIframe) {
                        console.log('Wheel event detected in iframe');
                        event.preventDefault();  // Prevent iframe scroll
                        event.stopImmediatePropagation();  // Prevent parent from scrolling
                }
            }, { passive: false });  // passive: false allows event.preventDefault()
        };
        console.log("Iframe found:", iframe);

        iframe.addEventListener("load", () => {
            iframe.addEventListener("mouseenter", event=> {
                isHoveringIframe = true;
            })

            iframe.addEventListener("mouseleave", event=> {
                isHoveringIframe = false;
            })

            iframe.addEventListener("wheel", event => {
                console.log("kill me");
                event.stopPropagation(); // Prevents parent scroll
            });
        });
    });
});
window.addEventListener("wheel", event => {
        console.log("wheel event detected in window");
        // event.preventDefault();
}, { passive: false }); // passive: false allows event.preventDefault()