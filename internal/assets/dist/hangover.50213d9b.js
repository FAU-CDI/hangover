var e="undefined"!=typeof globalThis?globalThis:"undefined"!=typeof self?self:"undefined"!=typeof window?window:"undefined"!=typeof global?global:{},t={},r={},n=e.parcelRequireafa4;null==n&&((n=function(e){if(e in t)return t[e].exports;if(e in r){var n=r[e];delete r[e];var l={id:e,exports:{}};return t[e]=l,n.call(l.exports,l,l.exports),l.exports}var i=Error("Cannot find module '"+e+"'");throw i.code="MODULE_NOT_FOUND",i}).register=function(e,t){r[e]=t},e.parcelRequireafa4=n),n.register("iLQcs",function(e,t){n("ibsO7"),n("b69nN")}),n.register("ibsO7",function(e,t){document.querySelectorAll(".tripletable").forEach(e=>{let t=Array.from(e.querySelectorAll("code.uri")),r=Array.from(e.querySelectorAll("tbody tr")),n=["violet","blue","red","indigo","green","orange"],l=0,i=new Map,o=()=>{t.forEach(e=>{let t=e.innerText,r=i.get(t);r?e.style.color=r:e.style.color=""})},a=()=>{i.clear(),o(),l=0},d=e=>{let t=e.innerText;console.log(e.innerText),i.has(t)?i.delete(t):(i.set(t,n[l]),l=(l+1)%n.length),o()},c=Array(r.length);r.forEach((e,t)=>{e.setAttribute("data-original-index",t.toString(10)),c[t]=new Set(Array.from(e.querySelectorAll("code.uri")).map(e=>e.innerText))});let u=()=>{r.forEach(e=>{let t=e.parentNode;t.removeChild(e),t.appendChild(e)})},s=()=>{r.sort((e,t)=>{let r=parseInt(e.getAttribute("data-original-index"),10),n=parseInt(t.getAttribute("data-original-index"),10);return r-n}),u()},f=e=>{let t=e.innerText;r.sort((e,r)=>{let n=parseInt(e.getAttribute("data-original-index"),10),l=parseInt(r.getAttribute("data-original-index"),10),i=c[n].has(t),o=c[l].has(t);return i===o?n-l:i?-1:1}),u(),e.scrollIntoView({behavior:"smooth"})};t.forEach(e=>{e.addEventListener("click",t=>{if(t.ctrlKey||t.metaKey){t.preventDefault(),d(e);return}if(t.altKey){t.preventDefault(),f(e);return}})}),e.querySelectorAll("thead tr").forEach(e=>e.addEventListener("click",e=>{!function(e){if(e.ctrlKey||e.metaKey){e.preventDefault(),a();return}e.altKey&&(e.preventDefault(),s())}(e)}))})}),n.register("b69nN",function(e,t){document.querySelectorAll(".showable").forEach(e=>{let t=e.querySelector(".toggle")??document.createElement("div");t.parentNode&&t.parentNode.removeChild(t),e.parentElement?.insertBefore(t,e);let r=e.style.display||"block",n=e.getAttribute("data-placeholder")??"Show",l=t.innerText||"Hide",i=i=>{i&&i.preventDefault(),t.innerText=o?l:n,o?(e.style.display=r,o=!1):(e.style.display="none",o=!0)},o=!1;t.addEventListener("click",i),i(null)})}),n("iLQcs");