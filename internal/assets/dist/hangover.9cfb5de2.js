var e,t,r,l;e="undefined"!=typeof globalThis?globalThis:"undefined"!=typeof self?self:"undefined"!=typeof window?window:"undefined"!=typeof global?global:{},t={},r={},null==(l=e.parcelRequireafa4)&&((l=function(e){if(e in t)return t[e].exports;if(e in r){var l=r[e];delete r[e];var n={id:e,exports:{}};return t[e]=n,l.call(n.exports,n,n.exports),n.exports}var o=Error("Cannot find module '"+e+"'");throw o.code="MODULE_NOT_FOUND",o}).register=function(e,t){r[e]=t},e.parcelRequireafa4=l),l.register("do6MR",function(e,t){l("7sWfK"),l("1kBYp")}),l.register("7sWfK",function(e,t){document.querySelectorAll(".tripletable").forEach(e=>{let t=Array.from(e.querySelectorAll("code.uri")),r=Array.from(e.querySelectorAll("tbody tr")),l=["violet","blue","red","indigo","green","orange"],n=0,o=new Map,a=()=>{t.forEach(e=>{let t=e.innerText,r=o.get(t);r?e.style.color=r:e.style.color=""})},i=()=>{o.clear(),a(),n=0},d=e=>{let t=e.innerText;console.log(e.innerText),o.has(t)?o.delete(t):(o.set(t,l[n]),n=(n+1)%l.length),a()},u=Array(r.length);r.forEach((e,t)=>{e.setAttribute("data-original-index",t.toString(10)),u[t]=new Set(Array.from(e.querySelectorAll("code.uri")).map(e=>e.innerText))});// function to rerender the rows
let c=()=>{r.forEach(e=>{let t=e.parentNode;t.removeChild(e),t.appendChild(e)})},f=()=>{r.sort((e,t)=>{let r=parseInt(e.getAttribute("data-original-index"),10),l=parseInt(t.getAttribute("data-original-index"),10);return r-l}),c()},s=e=>{let t=e.innerText;r.sort((e,r)=>{let l=parseInt(e.getAttribute("data-original-index"),10),n=parseInt(r.getAttribute("data-original-index"),10),o=u[l].has(t),a=u[n].has(t);return o===a?l-n:o?-1:1}),c(),e.scrollIntoView({behavior:"smooth"})};t.forEach(e=>{e.addEventListener("click",t=>{if(t.ctrlKey||t.metaKey){t.preventDefault(),d(e);return}if(t.altKey){t.preventDefault(),s(e);return}})}),e.querySelectorAll("thead tr").forEach(e=>e.addEventListener("click",e=>{!function(e){if(e.ctrlKey||e.metaKey){e.preventDefault(),i();return}e.altKey&&(e.preventDefault(),f())}(e)}))})}),l.register("1kBYp",function(e,t){document.querySelectorAll(".showable").forEach(e=>{// get or create a toggle
let t=e.querySelector(".toggle")??document.createElement("div");t.parentNode&&t.parentNode.removeChild(t),e.parentElement?.insertBefore(t,e);let r=e.style.display||"block",l=e.getAttribute("data-placeholder")??"Show",n=t.innerText||"Hide",o=o=>{o&&o.preventDefault(),t.innerText=a?n:l,a?(e.style.display=r,a=!1):(e.style.display="none",a=!0)},a=!1;t.addEventListener("click",o),o(null)})}),l("do6MR");