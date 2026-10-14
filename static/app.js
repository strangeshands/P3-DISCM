const $ = selector => document.querySelector(selector);
let library = [];
let uploading = false;
const formatSize = bytes => `${(bytes / 1024 / 1024).toFixed(1)} MB`;
function setStatus(message, error = false) {
  $('#status').textContent = message;
  $('#status').className = error ? 'error' : '';
}
async function refresh() {
  const response = await fetch('/api/videos');
  if (!response.ok) throw new Error('Could not load your library.');
  library = await response.json();
  render();
}
function render() {
  $('#count').textContent = library.length;
  $('#badge').textContent = library.length;
  const query = $('#search').value.toLowerCase();
  const filtered = library.filter(item => item.name.toLowerCase().includes(query));
  $('#gallery').replaceChildren();
  $('#empty').hidden = filtered.length > 0;
  $('#empty h3').textContent = library.length ? 'No matching videos.' : 'Your first video belongs here.';
  $('#empty p').textContent = library.length ? 'Try searching for another filename.' : 'Choose a video above to start building your library.';
  for (const item of filtered) {
    const card = document.createElement('button');
    card.className = 'card';
    card.setAttribute('aria-label', `Play ${item.name}`);
    const frame = document.createElement('div');
    frame.className = 'thumbnail';
    const video = document.createElement('video');
    video.src = item.url;
    video.preload = 'none';
    video.muted = true;
    video.playsInline = true;
    const overlay = document.createElement('span');
    overlay.className = 'play'; overlay.textContent = '▶';
    const hint = document.createElement('span');
    hint.className = 'preview-label'; hint.textContent = '10 SEC PREVIEW';
    frame.append(video, overlay, hint);
    const title = document.createElement('h3'); title.textContent = item.name;
    const meta = document.createElement('p');
    meta.textContent = `${formatSize(item.size)} · ${new Date(item.uploaded * 1000).toLocaleDateString()}`;
    card.append(frame, title, meta);
    let active = false;
    const stop = () => { active = false; video.pause(); video.currentTime = 0; card.classList.remove('previewing'); };
    card.addEventListener('mouseenter', () => {
      active = true;
      video.currentTime = 0;
      card.classList.add('previewing');
      video.play().then(() => { if (!active) video.pause(); }).catch(() => card.classList.remove('previewing'));
    });
    card.addEventListener('mouseleave', stop);
    video.addEventListener('timeupdate', () => { if (video.currentTime >= 10) stop(); });
    video.addEventListener('ended', stop);
    card.addEventListener('click', () => {
      stop();
      $('#player-title').textContent = item.name;
      $('#playback-error').textContent = '';
      $('#full-video').src = item.url;
      $('#player').showModal();
      $('#full-video').play().catch(() => {});
    });
    $('#gallery').append(card);
  }
}
async function upload(files) {
  if (uploading || !files.length) return;
  uploading = true; $('#choose').disabled = true;
  let saved = 0;
  const errors = [];
  for (const file of files) {
    setStatus(`Uploading ${saved + errors.length + 1} of ${files.length}: ${file.name}…`);
    try {
      const response = await fetch('/api/videos', {
        method: 'POST', headers: {'X-Filename': encodeURIComponent(file.name), 'Content-Type': 'application/octet-stream'}, body: file
      });
      const result = await response.json();
      if (!response.ok) throw new Error(result.error || 'Upload failed.');
      saved++;
    } catch (error) { errors.push(`${file.name}: ${error.message}`); }
  }
  try { await refresh(); } catch (error) { errors.push(error.message); }
  setStatus(`${saved} video${saved === 1 ? '' : 's'} saved.${errors.length ? ' ' + errors.join(' ') : ''}`, errors.length > 0);
  uploading = false; $('#choose').disabled = false; $('#files').value = '';
}
$('#choose').addEventListener('click', () => $('#files').click());
$('#files').addEventListener('change', event => upload([...event.target.files]));
$('#search').addEventListener('input', render);
$('#dropzone').addEventListener('dragover', event => { event.preventDefault(); $('#dropzone').classList.add('dragging'); });
$('#dropzone').addEventListener('dragleave', () => $('#dropzone').classList.remove('dragging'));
$('#dropzone').addEventListener('drop', event => { event.preventDefault(); $('#dropzone').classList.remove('dragging'); upload([...event.dataTransfer.files]); });
$('#close').addEventListener('click', () => $('#player').close());
$('#player').addEventListener('close', () => { $('#full-video').pause(); $('#full-video').removeAttribute('src'); $('#full-video').load(); });
$('#full-video').addEventListener('error', () => { $('#playback-error').textContent = 'Your browser cannot play this video codec. Try an H.264 MP4 or WebM video.'; });
refresh().catch(error => setStatus(error.message, true));
