!function(){var e="undefined"!=typeof globalThis?globalThis:"undefined"!=typeof self?self:"undefined"!=typeof window?window:"undefined"!=typeof global?global:{},n={},r={},o=e.parcelRequireafa4;null==o&&((o=function(e){if(e in n)return n[e].exports;if(e in r){var o=r[e];delete r[e];var t={id:e,exports:{}};return n[e]=t,o.call(t.exports,t,t.exports),t.exports}var l=Error("Cannot find module '"+e+"'");throw l.code="MODULE_NOT_FOUND",l}).register=function(e,n){r[e]=n},e.parcelRequireafa4=o),o("do6MR");let t=document.getElementById("progress"),l=0,a=null;async function i(){l++;let e=l;try{let n=await fetch("/api/v1/progress").then(e=>e.json());if(e!==l){console.warn("Received out-of-order response");return}// we are done loading
if(n.Done){t.innerHTML="Finished, reloading page ...",null!==a&&clearInterval(a),location.reload();return}if(0!==n.Total){let e=`<code>${n.Stage}</code> (<code>`;n.Total!==n.Current?e+=`${n.Current}/${n.Total}`:e+=n.Current.toString(),e+="</code>)",t.innerHTML=e}else""!=n.Stage&&(t.innerHTML=`<code>${n.Stage}</code>`)}catch(e){console.error(e)}}i(),null===a&&(a=setInterval(i,500))}();