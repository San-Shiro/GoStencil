// GoStencil WASM UI — app.js
// All rendering happens client-side via WASM. No server needed.

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

    // In-memory asset registry (mirrors Go-side assets).
    const jsAssets = {};

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
                        { type: "bullet", text: "100% client-side (WASM)" },
                        { type: "bullet", text: "Live preview as you type" },
                        { type: "bullet", text: "Export to PNG or AVI" },
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

        // Wait for WASM to be ready before first render
        function tryFirstRender() {
            if (window.goReady) { render(); }
            else { setTimeout(tryFirstRender, 100); }
        }
        tryFirstRender();
    }

    // Rendering (calls WASM directly)

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

    function render() {
        if (isRendering || !window.goReady) return;
        const parsed = getEditorJSON();
        if (parsed.error) { showError(parsed.error); return; }
        hideError();
        isRendering = true;
        previewLoading.classList.add('active');

        // Run in a microtask to avoid blocking UI
        setTimeout(() => {
            try {
                const result = window.goRenderImage(
                    JSON.stringify(parsed.preset),
                    JSON.stringify(parsed.data)
                );

                if (typeof result === 'string' && result.startsWith('error:')) {
                    showError(result.substring(6));
                } else {
                    // result is base64-encoded PNG
                    if (previewImg.src && previewImg.src.startsWith('blob:')) URL.revokeObjectURL(previewImg.src);
                    previewImg.src = 'data:image/png;base64,' + result;
                    previewImg.style.opacity = '1';
                }
            } catch (e) {
                showError('Render failed: ' + e.message);
            } finally {
                isRendering = false;
                previewLoading.classList.remove('active');
            }
        }, 10);
    }

    // Import (.gspresets is a ZIP — handle client-side)

    async function handleImport(e) {
        const file = e.target.files[0]; if (!file) return;
        e.target.value = '';

        try {
            const arrayBuf = await file.arrayBuffer();
            const entries = await readZip(new Uint8Array(arrayBuf));

            let presetJSON = null;
            for (const entry of entries) {
                if (entry.name === 'preset.json') {
                    presetJSON = new TextDecoder().decode(entry.data);
                } else if (entry.name.startsWith('assets/')) {
                    // Import asset
                    const assetName = entry.name.split('/').pop();
                    if (!assetName) continue;
                    const ext = assetName.split('.').pop().toLowerCase();
                    const mime = ext === 'ttf' ? 'font/ttf' : (ext === 'png' ? 'image/png' : (ext === 'jpg' || ext === 'jpeg' ? 'image/jpeg' : 'application/octet-stream'));
                    const id = assetName.replace(/\.[^.]+$/, ''); // use filename without ext as ID
                    registerAssetInBoth(id, entry.data, mime, assetName);
                }
            }

            if (presetJSON) {
                const presetObj = JSON.parse(presetJSON);
                presetEditor.value = JSON.stringify(presetObj, null, 2);
                dataEditor.value = JSON.stringify(buildDataTemplate(presetObj), null, 2);
                toast('Imported: ' + file.name, 'success');
                refreshAssetCount();
                render();
            } else {
                toast('No preset.json found in archive', 'error');
            }
        } catch (err) {
            toast('Import failed: ' + err.message, 'error');
        }
    }

    // ZIP reader — supports Stored (method 0) and Deflate (method 8)
    async function readZip(data) {
        const entries = [];
        const view = new DataView(data.buffer, data.byteOffset, data.byteLength);
        let pos = 0;

        while (pos < data.length - 4) {
            const sig = view.getUint32(pos, true);
            if (sig !== 0x04034b50) break; // Local file header signature

            const compMethod = view.getUint16(pos + 8, true);
            const compSize = view.getUint32(pos + 18, true);
            const uncompSize = view.getUint32(pos + 22, true);
            const nameLen = view.getUint16(pos + 26, true);
            const extraLen = view.getUint16(pos + 28, true);
            const name = new TextDecoder().decode(data.subarray(pos + 30, pos + 30 + nameLen));
            const dataStart = pos + 30 + nameLen + extraLen;
            const rawData = data.slice(dataStart, dataStart + compSize);

            if (compMethod === 0) {
                // Stored — raw data
                entries.push({ name, data: rawData });
            } else if (compMethod === 8) {
                // Deflate — decompress using browser API
                try {
                    const decompressed = await inflateRaw(rawData);
                    entries.push({ name, data: new Uint8Array(decompressed) });
                } catch (err) {
                    console.warn('Failed to decompress:', name, err);
                }
            }
            pos = dataStart + compSize;
        }
        return entries;
    }

    // Decompress raw deflate data using DecompressionStream API
    async function inflateRaw(compressedData) {
        const ds = new DecompressionStream('raw');
        const writer = ds.writable.getWriter();
        const reader = ds.readable.getReader();
        writer.write(compressedData);
        writer.close();
        const chunks = [];
        while (true) {
            const { done, value } = await reader.read();
            if (done) break;
            chunks.push(value);
        }
        const totalLen = chunks.reduce((s, c) => s + c.length, 0);
        const result = new Uint8Array(totalLen);
        let offset = 0;
        for (const c of chunks) {
            result.set(c, offset);
            offset += c.length;
        }
        return result.buffer;
    }

    // ZIP writer — creates a minimal ZIP with Stored entries (no compression)
    function createZip(files) {
        // files: [{ name: string, data: Uint8Array }]
        const centralDir = [];
        const parts = [];
        let offset = 0;

        for (const file of files) {
            const nameBytes = new TextEncoder().encode(file.name);
            const crc = crc32(file.data);

            // Local file header (30 + nameLen)
            const localHeader = new ArrayBuffer(30 + nameBytes.length);
            const lv = new DataView(localHeader);
            lv.setUint32(0, 0x04034b50, true);  // signature
            lv.setUint16(4, 20, true);            // version needed
            lv.setUint16(8, 0, true);             // compression: stored
            lv.setUint32(14, crc, true);          // crc32
            lv.setUint32(18, file.data.length, true); // compressed size
            lv.setUint32(22, file.data.length, true); // uncompressed size
            lv.setUint16(26, nameBytes.length, true);
            new Uint8Array(localHeader, 30).set(nameBytes);

            // Central directory entry
            const cdEntry = new ArrayBuffer(46 + nameBytes.length);
            const cv = new DataView(cdEntry);
            cv.setUint32(0, 0x02014b50, true);   // signature
            cv.setUint16(4, 20, true);             // version made by
            cv.setUint16(6, 20, true);             // version needed
            cv.setUint16(12, 0, true);             // compression: stored
            cv.setUint32(16, crc, true);
            cv.setUint32(20, file.data.length, true);
            cv.setUint32(24, file.data.length, true);
            cv.setUint16(28, nameBytes.length, true);
            cv.setUint32(42, offset, true);        // local header offset
            new Uint8Array(cdEntry, 46).set(nameBytes);
            centralDir.push(new Uint8Array(cdEntry));

            parts.push(new Uint8Array(localHeader));
            parts.push(file.data);
            offset += localHeader.byteLength + file.data.length;
        }

        // Central directory
        const cdOffset = offset;
        let cdSize = 0;
        for (const cd of centralDir) {
            parts.push(cd);
            cdSize += cd.length;
        }

        // End of central directory
        const eocd = new ArrayBuffer(22);
        const ev = new DataView(eocd);
        ev.setUint32(0, 0x06054b50, true);
        ev.setUint16(8, files.length, true);  // entries on disk
        ev.setUint16(10, files.length, true); // total entries
        ev.setUint32(12, cdSize, true);
        ev.setUint32(16, cdOffset, true);
        parts.push(new Uint8Array(eocd));

        // Combine
        const totalLen = parts.reduce((s, p) => s + p.length, 0);
        const result = new Uint8Array(totalLen);
        let pos = 0;
        for (const p of parts) {
            result.set(p, pos);
            pos += p.length;
        }
        return result;
    }

    // CRC-32 (used by ZIP format)
    function crc32(data) {
        if (!crc32.table) {
            crc32.table = new Uint32Array(256);
            for (let i = 0; i < 256; i++) {
                let c = i;
                for (let j = 0; j < 8; j++) c = (c & 1) ? (0xEDB88320 ^ (c >>> 1)) : (c >>> 1);
                crc32.table[i] = c;
            }
        }
        let crc = 0xFFFFFFFF;
        for (let i = 0; i < data.length; i++) crc = crc32.table[(crc ^ data[i]) & 0xFF] ^ (crc >>> 8);
        return (crc ^ 0xFFFFFFFF) >>> 0;
    }

    // Upload

    async function handleUploadFont(e) {
        const file = e.target.files[0]; if (!file) return;
        e.target.value = '';
        const data = new Uint8Array(await file.arrayBuffer());
        const id = randomId();
        registerAssetInBoth(id, data, 'font/ttf', file.name);

        try {
            const preset = JSON.parse(presetEditor.value);
            preset.font = { path: id };
            presetEditor.value = JSON.stringify(preset, null, 2);
            toast('Font loaded as global: ' + file.name, 'success');
            refreshAssetCount(); render();
        } catch (err) {
            toast('Font uploaded but preset update failed: ' + err.message, 'error');
        }
    }

    async function handleUploadImage(e) {
        const file = e.target.files[0]; if (!file) return;
        e.target.value = '';
        const data = new Uint8Array(await file.arrayBuffer());
        const mime = file.type || 'image/png';
        const id = randomId();
        registerAssetInBoth(id, data, mime, file.name);
        toast('Image uploaded: ' + file.name + ' - Open Assets to use it', 'success');
        refreshAssetCount();
    }

    // Register asset in both JS memory and Go WASM memory
    function registerAssetInBoth(id, uint8Data, mime, name) {
        // Store in JS
        jsAssets[id] = { name, data: uint8Data, mime, size: uint8Data.length };

        // Send to Go WASM as base64
        const b64 = uint8ArrayToBase64(uint8Data);
        const result = window.goRegisterAsset(id, b64, mime);
        if (result !== 'ok') {
            console.warn('goRegisterAsset failed:', result);
        }
    }

    function randomId() {
        const arr = new Uint8Array(8);
        crypto.getRandomValues(arr);
        return Array.from(arr).map(b => b.toString(16).padStart(2, '0')).join('');
    }

    function uint8ArrayToBase64(bytes) {
        let binary = '';
        for (let i = 0; i < bytes.length; i++) {
            binary += String.fromCharCode(bytes[i]);
        }
        return btoa(binary);
    }

    // Export

    function toggleExportMenu(e) { e.stopPropagation(); exportMenu.classList.toggle('open'); }

    function handleExport(type) {
        exportMenu.classList.remove('open');
        const parsed = getEditorJSON();
        if (parsed.error) { toast(parsed.error, 'error'); return; }
        switch (type) {
            case 'png':
                exportPNG(parsed);
                break;
            case 'avi':
                modalAvi.style.display = 'flex';
                break;
            case 'preset-json':
                downloadBlob(new Blob([JSON.stringify(parsed.preset, null, 2)], { type: 'application/json' }), 'preset.json');
                toast('Exported: preset.json', 'success');
                break;
            case 'data-json':
                downloadBlob(new Blob([JSON.stringify(parsed.data, null, 2)], { type: 'application/json' }), 'data.json');
                toast('Exported: data.json', 'success');
                break;
            case 'gspresets':
                exportGSPresets(parsed);
                break;
        }
    }

    function exportGSPresets(parsed) {
        try {
            const files = [];
            // Add preset.json
            const presetData = new TextEncoder().encode(JSON.stringify(parsed.preset, null, 2));
            files.push({ name: 'preset.json', data: presetData });
            // Add all assets
            for (const [id, asset] of Object.entries(jsAssets)) {
                const ext = mimeToExt(asset.mime);
                files.push({ name: 'assets/' + id + ext, data: asset.data });
            }
            const zipData = createZip(files);
            downloadBlob(new Blob([zipData], { type: 'application/zip' }), 'preset.gspresets');
            toast('Exported: preset.gspresets', 'success');
        } catch (e) {
            toast('Export failed: ' + e.message, 'error');
        }
    }

    function mimeToExt(mime) {
        if (!mime) return '';
        if (mime.indexOf('png') >= 0) return '.png';
        if (mime.indexOf('jpeg') >= 0 || mime.indexOf('jpg') >= 0) return '.jpg';
        if (mime.indexOf('webp') >= 0) return '.webp';
        if (mime.indexOf('ttf') >= 0 || mime.indexOf('font') >= 0) return '.ttf';
        if (mime.indexOf('otf') >= 0) return '.otf';
        return '';
    }

    function exportPNG(parsed) {
        try {
            const result = window.goRenderImage(
                JSON.stringify(parsed.preset),
                JSON.stringify(parsed.data)
            );
            if (typeof result === 'string' && result.startsWith('error:')) {
                toast('Export failed: ' + result, 'error');
                return;
            }
            const binary = atob(result);
            const bytes = new Uint8Array(binary.length);
            for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
            downloadBlob(new Blob([bytes], { type: 'image/png' }), 'output.png');
            toast('Exported: output.png', 'success');
        } catch (e) {
            toast('PNG export failed: ' + e.message, 'error');
        }
    }

    function doExportAVI() {
        modalAvi.style.display = 'none';
        const duration = parseInt($('#avi-duration').value) || 3;
        const parsed = getEditorJSON();
        if (parsed.error) { toast(parsed.error, 'error'); return; }
        toast('Generating AVI...', 'warn');

        setTimeout(() => {
            try {
                const result = window.goExportAVI(
                    JSON.stringify(parsed.preset),
                    JSON.stringify(parsed.data),
                    duration
                );
                if (typeof result === 'string' && result.startsWith('error:')) {
                    toast('AVI failed: ' + result, 'error');
                    return;
                }
                const binary = atob(result);
                const bytes = new Uint8Array(binary.length);
                for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
                downloadBlob(new Blob([bytes], { type: 'video/avi' }), 'output.avi');
                toast('Exported: output.avi', 'success');
            } catch (e) {
                toast('AVI export failed: ' + e.message, 'error');
            }
        }, 50);
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

    function refreshAssetCount() {
        assetCountEl.textContent = Object.keys(jsAssets).length;
    }

    function loadAssets() {
        const list = Object.entries(jsAssets).map(([id, a]) => ({
            id, name: a.name, mime: a.mime, size: a.size
        }));
        assetCountEl.textContent = list.length;
        renderAssetList(list);
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
            if (isImage) {
                // Create a blob URL for preview
                const blob = new Blob([jsAssets[a.id].data], { type: a.mime });
                const url = URL.createObjectURL(blob);
                previewHTML = '<img class="asset-card-preview" src="' + url + '" alt="' + a.name + '">';
            }

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

    function handleAssetAction(action, id, asset) {
        switch (action) {
            case 'copy-id':
                navigator.clipboard.writeText(id);
                toast('Copied ID: ' + id, 'success');
                break;

            case 'copy-bg-pair':
                var bgPair = '"backgroundImage": "' + id + '",\n"backgroundFit": "contain"';
                navigator.clipboard.writeText(bgPair);
                toast('Copied! Paste inside a component "style" object.', 'success');
                break;

            case 'copy-font-pair':
                var fontPair = '"fontPath": "' + id + '"';
                navigator.clipboard.writeText(fontPair);
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
                    } catch (de) { }

                    toast('Component "' + finalId + '" created! Adjust x/y/width/height.', 'success');
                    render();
                } catch (e) { toast('Could not create component: ' + e.message, 'error'); }
                break;

            case 'delete':
                // Remove from JS memory
                delete jsAssets[id];
                // Remove from Go WASM memory
                if (window.goRemoveAsset) window.goRemoveAsset(id);
                toast('Removed: ' + asset.name, 'success');
                loadAssets();
                render();
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
