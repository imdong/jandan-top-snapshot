// ==UserScript==
// @name         煎蛋快照 Release 图片包辅助
// @namespace    https://github.com/imdong/jandan-top-snapshot
// @version      1.0.0
// @description  跨域读取本仓库的 Release 图片包
// @match        https://page.qs5.org/jandan-top-snapshot/*
// @match        https://imdong.github.io/jandan-top-snapshot/*
// @match        http://localhost:8000/*
// @grant        GM_xmlhttpRequest
// @connect      github.com
// @connect      release-assets.githubusercontent.com
// @connect      objects.githubusercontent.com
// @run-at       document-start
// ==/UserScript==
(function () {
  'use strict';
  function ready() { document.documentElement.dataset.releaseBridge = 'ready'; }
  if (document.documentElement) ready(); else new MutationObserver((_, observer) => { if (document.documentElement) { ready(); observer.disconnect(); } }).observe(document, {childList:true});
  window.addEventListener('message', event => {
    if (event.source !== window || event.origin !== location.origin || event.data?.type !== 'snapshot-zip-request') return;
    const {id, url} = event.data;
    if (typeof id !== 'string' || typeof url !== 'string' || !/^https:\/\/github\.com\/imdong\/jandan-top-snapshot\/releases\/download\/snapshot-\d{8}-\d{6}\/media-\d{8}-\d{6}\.zip$/.test(url)) return;
    const reply = data => window.postMessage({type:'snapshot-zip-result', id, ...data}, location.origin);
    GM_xmlhttpRequest({method:'GET', url, responseType:'arraybuffer', timeout:180000,
      onload: response => response.status === 200 ? reply({buffer:response.response}) : reply({error:'HTTP ' + response.status}),
      onerror: () => reply({error:'跨域请求失败'}), ontimeout: () => reply({error:'请求超时'})});
  });
})();
