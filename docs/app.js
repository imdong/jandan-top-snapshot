'use strict';
const $ = id => document.getElementById(id);
let entries = [], current, selected, generation = 0, blobs = new Map(), archiveID = '';
const status = text => $('status').textContent = text;
function mediaURL(raw) {
  try {
    const u = new URL(raw, 'https://jandan.net');
    if (!['https:', 'http:'].includes(u.protocol)) return '';
    if (/\.gif$/i.test(u.pathname)) u.pathname = '/large/' + u.pathname.split('/').pop().replace(/\.gif$/i, '.mp4');
    else if (!/\.(webp|mp4|webm)$/i.test(u.pathname)) u.pathname += '.webp';
    return u.href;
  } catch { return ''; }
}
function clearArchive() { for (const u of blobs.values()) URL.revokeObjectURL(u); blobs.clear(); archiveID = ''; }
function unpack(buffer) {
  const view = new DataView(buffer), bytes = new Uint8Array(buffer), files = new Map();
  let offset;
  let end = bytes.length - 22;
  for (; end >= Math.max(0, bytes.length - 65557); end--) if (view.getUint32(end, true) === 0x06054b50) break;
  if (end < 0 || view.getUint32(end, true) !== 0x06054b50) throw Error('无效 ZIP');
  offset = view.getUint32(end + 16, true);
  const count = view.getUint16(end + 10, true);
  for (let i = 0; i < count; i++) {
    if (view.getUint32(offset, true) !== 0x02014b50) throw Error('ZIP 目录损坏');
    const method = view.getUint16(offset + 10, true), size = view.getUint32(offset + 20, true);
    const length = view.getUint16(offset + 28, true), extra = view.getUint16(offset + 30, true), comment = view.getUint16(offset + 32, true);
    const name = new TextDecoder().decode(bytes.slice(offset + 46, offset + 46 + length));
    const local = view.getUint32(offset + 42, true);
    const start = local + 30 + view.getUint16(local + 26, true) + view.getUint16(local + 28, true);
    if (method !== 0 || start + size > bytes.length) throw Error('请使用本项目生成的无压缩 ZIP');
    files.set(name, bytes.slice(start, start + size)); offset += 46 + length + extra + comment;
  }
  return files;
}
function installArchive(buffer, id) {
  const files = unpack(buffer);
  for (const name of Object.values(current.media)) if (!files.has(name)) throw Error('图片包与当前快照不匹配或文件缺失');
  clearArchive();
  for (const [name, data] of files) {
    const type = name.endsWith('.mp4') ? 'video/mp4' : name.endsWith('.webm') ? 'video/webm' : 'image/webp';
    blobs.set(name, URL.createObjectURL(new Blob([data], {type})));
  }
  archiveID = id; status(`已加载图片包（${files.size} 个媒体文件）`); render();
}
function contentNode(content) {
  const container = document.createElement('div'); container.className = 'content';
  const doc = new DOMParser().parseFromString(String(content || ''), 'text/html');
  function append(node, target) {
    if (node.nodeType === Node.TEXT_NODE) { target.append(document.createTextNode(node.textContent)); return; }
    if (node.nodeType !== Node.ELEMENT_NODE) return;
    const tag = node.tagName.toLowerCase();
    if (['script','style','iframe','object','embed','svg','math'].includes(tag)) return;
    if (['img','video','source'].includes(tag)) {
      const url = mediaURL(node.getAttribute('src') || node.getAttribute('data-src') || ''); if (!url) return;
      const archived = archiveID === selected.id && blobs.get(current.media[url]);
      const media = document.createElement(/\.(mp4|webm)(\?|$)/i.test(url) ? 'video' : 'img');
      media.src = $('mode').value === 'release' && archived ? archived : url;
      media.referrerPolicy = 'no-referrer';
      if (media.tagName === 'VIDEO') { media.controls = true; media.loop = true; media.muted = true; media.playsInline = true; media.preload = 'none'; }
      else { media.loading = 'lazy'; media.alt = node.getAttribute('alt') || '快照图片'; }
      media.addEventListener('error', () => { const tip = document.createElement('p'); tip.textContent = '媒体加载失败，可切换来源或导入图片包。'; media.replaceWith(tip); }, {once:true});
      target.append(media); return;
    }
    const allowed = ['p','br','strong','b','em','i','blockquote','ul','ol','li','a','code','pre'];
    const el = document.createElement(allowed.includes(tag) ? tag : 'span');
    if (tag === 'a') {
      try { const u = new URL(node.getAttribute('href'), 'https://jandan.net'); if (['https:','http:'].includes(u.protocol)) { el.href = u.href; el.target = '_blank'; el.rel = 'noopener noreferrer'; } } catch {}
    }
    for (const child of node.childNodes) append(child, el); target.append(el);
  }
  for (const node of doc.body.childNodes) append(node, container);
  return container;
}
function render() {
  if (!current) return;
  $('posts').replaceChildren();
  for (const post of current.data) {
    const article = document.createElement('article'), meta = document.createElement('div'); meta.className = 'meta';
    const author = document.createElement('strong'); author.textContent = post.author;
    const link = document.createElement('a'); link.textContent = '#' + post.id; link.href = 'https://jandan.net/t/' + encodeURIComponent(post.id); link.target = '_blank'; link.rel = 'noopener';
    meta.append(author, link); article.append(meta, contentNode(post.content));
    const votes = document.createElement('div'); votes.className = 'votes'; votes.textContent = `OO ${post.vote_positive ?? 0}　XX ${post.vote_negative ?? 0}　吐槽 ${post.sub_comment_count ?? 0}`; article.append(votes);
    if (post.hot_tucao) { const hot = document.createElement('div'); hot.className = 'hot'; const name = document.createElement('strong'); name.textContent = '神吐槽 · ' + post.hot_tucao.author; hot.append(name, contentNode(post.hot_tucao.content)); article.append(hot); }
    $('posts').append(article);
  }
}
async function loadArchive() {
  if (!current || $('mode').value !== 'release') return;
  if (archiveID === selected.id) { render(); return; }
  const id = selected.id, token = generation, url = selected.release;
  status('正在读取 Release 图片包…');
  try {
    let buffer;
    if (document.documentElement.dataset.releaseBridge === 'ready') {
      buffer = await new Promise((resolve, reject) => {
        const requestId = crypto.randomUUID();
        const timer = setTimeout(() => { window.removeEventListener('message', listener); reject(Error('读取超时')); }, 180000);
        function listener(event) { if (event.source !== window || event.origin !== location.origin || event.data?.type !== 'snapshot-zip-result' || event.data.id !== requestId) return; clearTimeout(timer); window.removeEventListener('message', listener); event.data.error ? reject(Error(event.data.error)) : resolve(event.data.buffer); }
        window.addEventListener('message', listener); window.postMessage({type:'snapshot-zip-request', id:requestId, url}, location.origin);
      });
    } else { const response = await fetch(url); if (!response.ok) throw Error('HTTP ' + response.status); buffer = await response.arrayBuffer(); }
    if (token !== generation) return;
    installArchive(buffer, id);
  } catch (error) { if (token !== generation) return; status('图片包读取失败：' + error.message + '。请安装辅助脚本并刷新，或下载 ZIP 后导入；当前暂用原站图片。'); render(); }
}
async function selectSnapshot() {
  const token = ++generation; clearArchive(); current = null; $('posts').replaceChildren(); selected = entries.find(e => e.id === $('snapshot').value);
  $('download').hidden = !selected; if (!selected) return;
  $('download').href = selected.release; status('正在加载快照…');
  try { const response = await fetch(selected.file); if (!response.ok) throw Error('HTTP ' + response.status); const data = await response.json(); if (token !== generation) return; if (!Array.isArray(data.data)) throw Error('无效快照'); current = data; history.replaceState(null, '', '#' + selected.id); status(`${selected.time} · ${current.data.length} 条内容`); render(); await loadArchive(); }
  catch (error) { if (token === generation) status('快照加载失败：' + error.message); }
}
$('snapshot').addEventListener('change', selectSnapshot);
$('mode').addEventListener('change', () => { render(); loadArchive(); });
$('archive').addEventListener('change', async event => { const file = event.target.files[0], token = generation; if (!file || !current) return; try { const buffer = await file.arrayBuffer(); if (token !== generation) return; $('mode').value = 'release'; installArchive(buffer, selected.id); } catch (error) { status(error.message); } event.target.value = ''; });
(async () => { try { const response = await fetch('snapshots.json'); if (!response.ok) throw Error('HTTP ' + response.status); entries = await response.json(); for (const entry of entries) { const option = document.createElement('option'); option.value = entry.id; option.textContent = entry.time; $('snapshot').append(option); } if (!entries.length) { status('暂无 JSON 快照，请运行采集任务。'); return; } if (entries.some(e => e.id === location.hash.slice(1))) $('snapshot').value = location.hash.slice(1); await selectSnapshot(); } catch (error) { status('索引读取失败：' + error.message); } })();
