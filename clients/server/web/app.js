// GoStencil UI - app.js

(() => {
  'use strict';

  const $ = (s) => document.querySelector(s);
  const presetEditor = $('#editor-preset');
  const dataEditor = $('#editor-data');
  const previewImg = $('#preview-img');
  const previewLoading = $('#preview-loading');
  const previewError = $('#preview-error');
  const exportMenu = $('#export-menu');
  const toastContainer = $('#toasts');
  const modalAvi = $('#modal-avi');
  const assetPanel = $('#asset-panel');
  const assetBackdrop = $('#asset-backdrop');
  const assetList = $('#asset-list');
  const assetCountEl = $('#asset-count');

  // Default JSON
  const defaultPreset = {
    meta: { name: "My Preset", version: "1.0", author: "Author", description: "" },
    canvas: { preset: "1080p" },
    background: { type: "color", color: "#1a1a2e" },
    font: {},
    components: [
      {
        id: "title", x: 0.05, y: 0.05, width: 0.9, height: 0.2, zIndex: 1, padding: 30,
        style: { backgroundColor: "#ffffff10", cornerRadius: 12, fontSize: 48, color: "#ffffff", lineHeight: 1.3, textAlign: "center" },
        defaults: { visible: true, title: "Hello, GoStencil!", items: [{ type: "text", text: "Edit this preset to get started" }] }
      },
      {
        id: "content", x: 0.05, y: 0.3, width: 0.9, height: 0.5, zIndex: 2, padding: 25,
        style: { backgroundColor: "#ffffff08", cornerRadius: 10, fontSize: 24, color: "#cccccc", lineHeight: 1.6, textAlign: "left" },
        defaults: {
          visible: true, title: "Features", items: [
            { type: "bullet", text: "Live preview as you type" },
            { type: "bullet", text: "Import .gspresets bundles" },
            { type: "bullet", text: "Export to PNG, AVI, or .gspresets" },
            { type: "bullet", text: "Custom fonts and images" }
          ]
        }
      }
    ],
    schema: {
      description: "Default preset schema", components: {
        title: { description: "Main title area", fields: { visible: "boolean", title: "string", items: "array" } },
        content: { description: "Content area", fields: { visible: "boolean", title: "string", items: "array" } }
      }
    }
  };

  // Build data.json template with commented-out overrides.
  // Keys prefixed with "// " are ignored by the Go parser.
  // To activate an override, just remove the "// " prefix from the key.
  function buildDataTemplate(preset) {
    var comps = {};
    if (preset && preset.components) {
      for (var i = 0; i < preset.components.length; i++) {
        var c = preset.components[i];
        var entry = {};
        entry.visible = true;
        if (c.defaults && c.defaults.title) entry.title = c.defaults.title;
        if (c.defaults && c.defaults.items && c.defaults.items.length > 0) entry.items = c.defaults.items;
        entry.style = {};
        if (c.style) {
          if (c.style.fontSize) entry.style.fontSize = c.style.fontSize;
          if (c.style.color) entry.style.color = c.style.color;
          if (c.style.textAlign) entry.style.textAlign = c.style.textAlign;
        }
        // Prefix with "// " so it is commented out (ignored by Go).
        // Remove the "// " prefix to activate overrides for this component.
        comps['// ' + c.id] = entry;
      }
    }
    return { components: comps };
  }

  // State
  let renderTimeout = null;
  let isRendering = false;

  // Init
  function init() {
    presetEditor.value = JSON.stringify(defaultPreset, null, 2);
    dataEditor.value = JSON.stringify(buildDataTemplate(defaultPreset), null, 2);

    presetEditor.addEventListener('input', scheduleRender);
    dataEditor.addEventListener('input', scheduleRender);
    presetEditor.addEventListener('keydown', handleTab);
    dataEditor.addEventListener('keydown', handleTab);

    $('#btn-import').addEventListener('click', () => $('#file-import').click());
    $('#btn-upload-font').addEventListener('click', () => $('#file-font').click());
    $('#btn-upload-image').addEventListener('click', () => $('#file-image').click());
    $('#btn-export').addEventListener('click', toggleExportMenu);
    $('#btn-assets').addEventListener('click', openAssetPanel);
    $('#btn-close-assets').addEventListener('click', closeAssetPanel);
    $('#btn-help').addEventListener('click', () => $('#modal-help').style.display = 'flex');
    $('#help-close').addEventListener('click', () => $('#modal-help').style.display = 'none');
    assetBackdrop.addEventListener('click', closeAssetPanel);

    $('#file-import').addEventListener('change', handleImport);
    $('#file-font').addEventListener('change', handleUploadFont);
    $('#file-image').addEventListener('change', handleUploadImage);

    exportMenu.querySelectorAll('[data-export]').forEach(btn => {
      btn.addEventListener('click', () => handleExport(btn.dataset.export));
    });

    document.addEventListener('click', (e) => {
      if (!e.target.closest('.export-group')) exportMenu.classList.remove('open');
    });

    $('#btn-zoom-fit').addEventListener('click', () => setZoom('fit'));
    $('#btn-zoom-100').addEventListener('click', () => setZoom('100'));
    $('#avi-cancel').addEventListener('click', () => modalAvi.style.display = 'none');
    $('#avi-export').addEventListener('click', doExportAVI);

    initResize();
    render();
  }

  // Rendering

  function scheduleRender() {
    clearTimeout(renderTimeout);
    renderTimeout = setTimeout(render, 350);
  }

  function getEditorJSON() {
    let preset, data;
    try { preset = JSON.parse(presetEditor.value); } catch (e) { return { error: 'Preset JSON: ' + e.message }; }
    try { data = JSON.parse(dataEditor.value); } catch (e) { return { error: 'Data JSON: ' + e.message }; }
    return { preset, data };
  }

  async function render() {
    if (isRendering) return;
    const parsed = getEditorJSON();
    if (parsed.error) { showError(parsed.error); return; }
    hideError();
    isRendering = true;
    previewLoading.classList.add('active');

    try {
      const res = await fetch('/api/render', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ preset: parsed.preset, data: parsed.data })
      });
      if (!res.ok) { showError(await res.text()); return; }
      const blob = await res.blob();
      const url = URL.createObjectURL(blob);
      if (previewImg.src && previewImg.src.startsWith('blob:')) URL.revokeObjectURL(previewImg.src);
      previewImg.src = url;
      previewImg.style.opacity = '1';
    } catch (e) {
      showError('Render failed: ' + e.message);
    } finally {
      isRendering = false;
      previewLoading.classList.remove('active');
    }
  }

  // Import

  async function handleImport(e) {
    const file = e.target.files[0]; if (!file) return;
    e.target.value = '';
    const form = new FormData(); form.append('file', file);
    try {
      const res = await fetch('/api/import/gspresets', { method: 'POST', body: form });
      if (!res.ok) throw new Error(await res.text());
      const result = await res.json();
      let presetObj = typeof result.preset === 'string' ? JSON.parse(result.preset) : result.preset;
      presetEditor.value = JSON.stringify(presetObj, null, 2);
      dataEditor.value = JSON.stringify(buildDataTemplate(presetObj), null, 2);
      toast('Imported: ' + file.name, 'success');
      refreshAssetCount(); render();
    } catch (err) { toast('Import failed: ' + err.message, 'error'); }
  }

  // Upload

  async function handleUploadFont(e) {
    const file = e.target.files[0]; if (!file) return;
    e.target.value = '';
    const form = new FormData(); form.append('file', file);
    try {
      const res = await fetch('/api/upload/font', { method: 'POST', body: form });
      if (!res.ok) throw new Error(await res.text());
      const result = await res.json();
      try {
        const preset = JSON.parse(presetEditor.value);
        preset.font = { path: result.id };
        presetEditor.value = JSON.stringify(preset, null, 2);
        toast('Font loaded as global: ' + result.name, 'success');
        refreshAssetCount(); render();
      } catch (err) { toast('Font uploaded but preset update failed: ' + err.message, 'error'); }
    } catch (err) { toast('Font upload failed: ' + err.message, 'error'); }
  }

  async function handleUploadImage(e) {
    const file = e.target.files[0]; if (!file) return;
    e.target.value = '';
    const form = new FormData(); form.append('file', file);
    try {
      const res = await fetch('/api/upload/image', { method: 'POST', body: form });
      if (!res.ok) throw new Error(await res.text());
      const result = await res.json();
      toast('Image uploaded: ' + result.name + ' - Open Assets to use it', 'success');
      refreshAssetCount();
    } catch (err) { toast('Image upload failed: ' + err.message, 'error'); }
  }

  // Export

  function toggleExportMenu(e) { e.stopPropagation(); exportMenu.classList.toggle('open'); }

  function handleExport(type) {
    exportMenu.classList.remove('open');
    const parsed = getEditorJSON();
    if (parsed.error) { toast(parsed.error, 'error'); return; }
    switch (type) {
      case 'png':
        downloadFromAPI('/api/export/png', { preset: parsed.preset, data: parsed.data }, 'output.png');
        break;
      case 'avi':
        modalAvi.style.display = 'flex';
        break;
      case 'preset-json':
        // Direct client-side download for JSON - no server round-trip needed
        downloadBlob(new Blob([JSON.stringify(parsed.preset, null, 2)], { type: 'application/json' }), 'preset.json');
        toast('Exported: preset.json', 'success');
        break;
      case 'data-json':
        downloadBlob(new Blob([JSON.stringify(parsed.data, null, 2)], { type: 'application/json' }), 'data.json');
        toast('Exported: data.json', 'success');
        break;
      case 'gspresets':
        downloadFromAPI('/api/export/gspresets', { preset: parsed.preset }, 'preset.gspresets');
        break;
    }
  }

  async function doExportAVI() {
    modalAvi.style.display = 'none';
    const duration = parseInt($('#avi-duration').value) || 3;
    const parsed = getEditorJSON();
    if (parsed.error) { toast(parsed.error, 'error'); return; }
    toast('Generating AVI...', 'warn');
    await downloadFromAPI('/api/export/avi', { preset: parsed.preset, data: parsed.data, duration: duration }, 'output.avi');
  }

  async function downloadFromAPI(url, body, filename) {
    try {
      const res = await fetch(url, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body)
      });
      if (!res.ok) {
        const errText = await res.text();
        throw new Error(errText || 'Server returned ' + res.status);
      }
      const blob = await res.blob();
      if (blob.size === 0) {
        throw new Error('Server returned empty response');
      }
      downloadBlob(blob, filename);
      toast('Exported: ' + filename, 'success');
    } catch (err) {
      toast('Export failed: ' + err.message, 'error');
    }
  }

  function downloadBlob(blob, filename) {
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    a.style.display = 'none';
    document.body.appendChild(a);
    a.click();
    setTimeout(() => {
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    }, 1000);
  }

  // Asset Manager

  function openAssetPanel() { assetPanel.classList.add('open'); assetBackdrop.classList.add('open'); loadAssets(); }
  function closeAssetPanel() { assetPanel.classList.remove('open'); assetBackdrop.classList.remove('open'); }

  async function refreshAssetCount() {
    try { const res = await fetch('/api/assets'); const a = await res.json(); assetCountEl.textContent = a.length; } catch (e) { }
  }

  async function loadAssets() {
    try {
      const res = await fetch('/api/assets');
      const assets = await res.json();
      assetCountEl.textContent = assets.length;
      renderAssetList(assets);
    } catch (e) { assetList.innerHTML = '<div class="asset-empty">Failed to load assets</div>'; }
  }

  function renderAssetList(assets) {
    if (!assets || assets.length === 0) {
      assetList.innerHTML = '<div class="asset-empty">No assets uploaded yet. Use Font or Image buttons to upload.</div>';
      return;
    }
    assetList.innerHTML = '';
    for (const a of assets) {
      const isImage = a.mime && a.mime.startsWith('image/');
      const isFont = a.mime && (a.mime.indexOf('font') >= 0 || a.mime.indexOf('ttf') >= 0 || a.mime.indexOf('otf') >= 0);
      const sizeKB = (a.size / 1024).toFixed(1);
      const typeLabel = isFont ? 'font' : (isImage ? 'image' : 'file');

      const card = document.createElement('div');
      card.className = 'asset-card';

      let previewHTML = '';
      if (isImage) previewHTML = '<img class="asset-card-preview" src="/api/assets/' + a.id + '" alt="' + a.name + '">';

      let actionsHTML = '<button class="asset-btn" data-action="copy-id" data-id="' + a.id + '" title="Copy asset ID">[ID] Copy ID</button>';

      if (isFont) {
        actionsHTML += '<button class="asset-btn" data-action="use-font-global" data-id="' + a.id + '" title="Set as global font">[Aa] Global Font</button>';
        actionsHTML += '<button class="asset-btn" data-action="copy-font-pair" data-id="' + a.id + '" title="Copy fontPath key-pair">[{}] Copy fontPath</button>';
      }
      if (isImage) {
        actionsHTML += '<button class="asset-btn" data-action="copy-bg-pair" data-id="' + a.id + '" title="Copy backgroundImage key-value pair">[{}] Copy BG Pair</button>';
        actionsHTML += '<button class="asset-btn" data-action="make-component" data-id="' + a.id + '" title="Create a new image component in preset">[+] Make Component</button>';
      }
      actionsHTML += '<button class="asset-btn danger" data-action="delete" data-id="' + a.id + '" title="Remove asset">[x] Remove</button>';

      card.innerHTML = '<div class="asset-card-header">'
        + '<span class="asset-card-name" title="' + a.name + '">' + a.name + '</span>'
        + '<span class="asset-card-type ' + typeLabel + '">' + typeLabel + '</span>'
        + '</div>'
        + '<div class="asset-card-meta">' + sizeKB + ' KB</div>'
        + previewHTML
        + '<div class="asset-card-id"><span>ID:</span> <code title="' + a.id + '">' + a.id + '</code></div>'
        + '<div class="asset-card-actions">' + actionsHTML + '</div>';

      card.querySelectorAll('[data-action]').forEach(btn => {
        btn.addEventListener('click', () => handleAssetAction(btn.dataset.action, btn.dataset.id, a));
      });
      assetList.appendChild(card);
    }
  }

  async function handleAssetAction(action, id, asset) {
    switch (action) {
      case 'copy-id':
        await navigator.clipboard.writeText(id);
        toast('Copied ID: ' + id, 'success');
        break;

      case 'copy-bg-pair':
        var bgPair = '"backgroundImage": "' + id + '",\n"backgroundFit": "contain"';
        await navigator.clipboard.writeText(bgPair);
        toast('Copied! Paste inside a component "style" object.', 'success');
        break;

      case 'copy-font-pair':
        var fontPair = '"fontPath": "' + id + '"';
        await navigator.clipboard.writeText(fontPair);
        toast('Copied! Paste inside a component "style" object.', 'success');
        break;

      case 'use-font-global':
        try {
          var preset = JSON.parse(presetEditor.value);
          preset.font = { path: id };
          presetEditor.value = JSON.stringify(preset, null, 2);
          toast('Global font set', 'success');
          render();
        } catch (e) { toast('Could not update preset: ' + e.message, 'error'); }
        break;

      case 'make-component':
        try {
          var p = JSON.parse(presetEditor.value);
          if (!p.components) p.components = [];
          var compId = 'img_' + id.substring(0, 6);
          var existingIds = {};
          for (var i = 0; i < p.components.length; i++) existingIds[p.components[i].id] = true;
          var finalId = compId;
          var n = 2;
          while (existingIds[finalId]) { finalId = compId + '_' + n; n++; }
          var maxZ = 0;
          for (var j = 0; j < p.components.length; j++) {
            if ((p.components[j].zIndex || 0) > maxZ) maxZ = p.components[j].zIndex || 0;
          }
          var newComp = {
            id: finalId,
            x: 0.1, y: 0.1, width: 0.3, height: 0.3,
            zIndex: maxZ + 1,
            padding: 0,
            style: {
              backgroundImage: id,
              backgroundFit: "contain",
              backgroundColor: "",
              fontSize: 1, color: "#ffffff", lineHeight: 1.5, textAlign: "left"
            },
            defaults: { visible: true, title: "", items: [] }
          };
          p.components.push(newComp);
          presetEditor.value = JSON.stringify(p, null, 2);

          // Also add a commented entry in data.json
          try {
            var d = JSON.parse(dataEditor.value);
            if (!d.components) d.components = {};
            d.components['// ' + finalId] = {
              visible: true,
              title: "",
              items: [],
              style: { fontSize: 1, color: "#ffffff", textAlign: "left" }
            };
            dataEditor.value = JSON.stringify(d, null, 2);
          } catch (de) { /* data parse failed, skip */ }

          toast('Component "' + finalId + '" created! Adjust x/y/width/height.', 'success');
          render();
        } catch (e) { toast('Could not create component: ' + e.message, 'error'); }
        break;

      case 'delete':
        try {
          var res = await fetch('/api/assets/' + id, { method: 'DELETE' });
          if (!res.ok) throw new Error(await res.text());
          toast('Removed: ' + asset.name, 'success');
          loadAssets(); render();
        } catch (e) { toast('Delete failed: ' + e.message, 'error'); }
        break;
    }
  }

  // Zoom

  function setZoom(mode) {
    var fitBtn = $('#btn-zoom-fit');
    var fullBtn = $('#btn-zoom-100');
    if (mode === '100') {
      previewImg.classList.add('zoom-100');
      fitBtn.classList.remove('active'); fullBtn.classList.add('active');
      $('#zoom-label').textContent = '100%';
    } else {
      previewImg.classList.remove('zoom-100');
      fitBtn.classList.add('active'); fullBtn.classList.remove('active');
      $('#zoom-label').textContent = 'Fit';
    }
  }

  // Resize Handles

  function initResize() {
    document.querySelectorAll('.resize-handle').forEach(function (handle) {
      handle.addEventListener('mousedown', function (e) {
        e.preventDefault();
        handle.classList.add('active');
        var startX = e.clientX;
        var lp, rp;
        if (handle.dataset.resize === 'left') { lp = $('#panel-preset'); rp = $('#panel-data'); }
        else { lp = $('#panel-data'); rp = $('#panel-preview'); }
        var slw = lp.offsetWidth, srw = rp.offsetWidth;
        function onMove(e) {
          var dx = e.clientX - startX;
          lp.style.flex = '0 0 ' + Math.max(200, slw + dx) + 'px';
          rp.style.flex = '0 0 ' + Math.max(200, srw - dx) + 'px';
        }
        function onUp() { handle.classList.remove('active'); document.removeEventListener('mousemove', onMove); document.removeEventListener('mouseup', onUp); }
        document.addEventListener('mousemove', onMove);
        document.addEventListener('mouseup', onUp);
      });
    });
  }

  // Tab key

  function handleTab(e) {
    if (e.key !== 'Tab') return;
    e.preventDefault();
    var ta = e.target;
    var s = ta.selectionStart;
    ta.value = ta.value.substring(0, s) + '  ' + ta.value.substring(ta.selectionEnd);
    ta.selectionStart = ta.selectionEnd = s + 2;
  }

  // Error / Toast

  function showError(msg) { previewError.textContent = msg; previewError.classList.add('active'); }
  function hideError() { previewError.classList.remove('active'); }

  function toast(message, type) {
    var el = document.createElement('div');
    el.className = 'toast' + (type ? ' toast--' + type : '');
    el.textContent = message;
    toastContainer.appendChild(el);
    setTimeout(function () {
      el.style.opacity = '0';
      el.style.transform = 'translateX(30px)';
      el.style.transition = '300ms';
      setTimeout(function () { el.remove(); }, 300);
    }, 3500);
  }

  document.addEventListener('DOMContentLoaded', init);
})();
