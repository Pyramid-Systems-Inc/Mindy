package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"mindy/internal/blob"
	"mindy/internal/dataman"
	"mindy/internal/graph"
	"mindy/internal/indexer"
	"mindy/internal/vector"
	"mindy/pkg/embedder"
)

const webUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Mindy - Personal AI Memory</title>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap" rel="stylesheet">
    <style>
        :root { --primary: #6366f1; --primary-dark: #4f46e5; --primary-light: #818cf8; --bg: #0f172a; --bg-secondary: #1e293b; --bg-tertiary: #334155; --text: #f8fafc; --text-secondary: #94a3b8; --text-muted: #64748b; --border: #334155; --success: #22c55e; --error: #ef4444; --card-bg: #1e293b; --hover: #334155; }
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; background: var(--bg); color: var(--text); min-height: 100vh; line-height: 1.6; }
        .app { max-width: 1400px; margin: 0 auto; padding: 2rem; }
        .header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 2rem; padding-bottom: 1.5rem; border-bottom: 1px solid var(--border); }
        .logo { display: flex; align-items: center; gap: 1rem; }
        .logo-icon { width: 48px; height: 48px; background: linear-gradient(135deg, var(--primary) 0%, var(--primary-light) 100%); border-radius: 12px; display: flex; align-items: center; justify-content: center; font-size: 1.5rem; font-weight: 700; }
        .logo h1 { font-size: 1.75rem; font-weight: 700; background: linear-gradient(135deg, var(--primary-light) 0%, var(--primary) 100%); -webkit-background-clip: text; -webkit-text-fill-color: transparent; }
        .logo p { color: var(--text-secondary); font-size: 0.875rem; }
        .stats-bar { display: flex; gap: 1rem; }
        .stat-item { background: var(--card-bg); padding: 0.5rem 1rem; border-radius: 8px; font-size: 0.875rem; color: var(--text-secondary); }
        .stat-item span { color: var(--primary-light); font-weight: 600; }
        .search-container { background: var(--card-bg); border-radius: 16px; padding: 1.5rem; margin-bottom: 1.5rem; border: 1px solid var(--border); }
        .search-form { display: flex; gap: 0.75rem; }
        .search-input-wrapper { position: relative; flex: 1; }
        .search-input { width: 100%; padding: 1rem 1.25rem; font-size: 1rem; background: var(--bg); border: 2px solid var(--border); border-radius: 12px; color: var(--text); transition: all 0.2s; }
        .search-input:focus { outline: none; border-color: var(--primary); box-shadow: 0 0 0 4px rgba(99, 102, 241, 0.1); }
        .search-input::placeholder { color: var(--text-muted); }
        .autocomplete-dropdown { position: absolute; top: calc(100% + 8px); left: 0; right: 0; background: var(--card-bg); border: 1px solid var(--border); border-radius: 12px; max-height: 320px; overflow-y: auto; z-index: 1000; box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.3); display: none; }
        .autocomplete-dropdown.show { display: block; }
        .autocomplete-item { padding: 0.875rem 1rem; cursor: pointer; border-bottom: 1px solid var(--border); display: flex; justify-content: space-between; align-items: center; transition: background 0.15s; }
        .autocomplete-item:hover { background: var(--hover); }
        .autocomplete-item .label { font-weight: 500; }
        .autocomplete-item .type { font-size: 0.75rem; color: var(--text-muted); background: var(--bg); padding: 0.25rem 0.5rem; border-radius: 4px; }
        .btn { padding: 0.875rem 1.5rem; font-size: 0.95rem; font-weight: 500; border: none; border-radius: 12px; cursor: pointer; transition: all 0.2s; display: inline-flex; align-items: center; gap: 0.5rem; }
        .btn-primary { background: linear-gradient(135deg, var(--primary) 0%, var(--primary-dark) 100%); color: white; }
        .btn-primary:hover { transform: translateY(-1px); box-shadow: 0 10px 15px -3px rgba(99, 102, 241, 0.3); }
        .btn-secondary { background: var(--bg-tertiary); color: var(--text); }
        .btn-secondary:hover { background: var(--hover); }
        .tabs { display: flex; gap: 0.5rem; margin-bottom: 1.5rem; flex-wrap: wrap; }
        .tab { padding: 0.75rem 1.25rem; background: transparent; border: none; cursor: pointer; font-size: 0.9rem; font-weight: 500; color: var(--text-secondary); border-radius: 8px; transition: all 0.2s; }
        .tab:hover { background: var(--hover); color: var(--text); }
        .tab.active { background: var(--primary); color: white; }
        .tab-content { display: none; }
        .tab-content.active { display: block; }
        .results-container { background: var(--card-bg); border-radius: 16px; padding: 1.5rem; border: 1px solid var(--border); }
        .results-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem; padding-bottom: 1rem; border-bottom: 1px solid var(--border); }
        .results-header h3 { font-size: 1.1rem; font-weight: 600; }
        .result-item { padding: 1rem; border-radius: 12px; margin-bottom: 0.75rem; cursor: pointer; transition: all 0.2s; background: var(--bg); border: 1px solid transparent; }
        .result-item:hover { background: var(--hover); border-color: var(--primary); transform: translateX(4px); }
        .result-item:last-child { margin-bottom: 0; }
        .result-score { display: inline-flex; align-items: center; gap: 0.5rem; background: linear-gradient(135deg, var(--primary) 0%, var(--primary-dark) 100%); color: white; padding: 0.25rem 0.75rem; border-radius: 20px; font-size: 0.75rem; font-weight: 600; margin-bottom: 0.5rem; }
        .result-path { display: block; font-weight: 500; color: var(--text); margin-bottom: 0.25rem; word-break: break-all; }
        .result-meta { font-size: 0.8rem; color: var(--text-muted); }
        .empty-state { text-align: center; padding: 3rem; color: var(--text-muted); }
        .empty-state svg { width: 64px; height: 64px; margin-bottom: 1rem; opacity: 0.5; }
        .loading { text-align: center; padding: 2rem; color: var(--text-secondary); }
        .spinner { width: 40px; height: 40px; border: 3px solid var(--border); border-top-color: var(--primary); border-radius: 50%; animation: spin 1s linear infinite; margin: 0 auto 1rem; }
        @keyframes spin { to { transform: rotate(360deg); } }
        .panel { background: var(--card-bg); border-radius: 12px; padding: 1.25rem; border: 1px solid var(--border); margin-bottom: 1rem; }
        .panel h3 { font-size: 0.85rem; font-weight: 600; color: var(--text-secondary); text-transform: uppercase; letter-spacing: 0.05em; margin-bottom: 1rem; }
        .history-item, .saved-item { padding: 0.75rem; cursor: pointer; border-radius: 8px; margin-bottom: 0.5rem; transition: all 0.15s; display: flex; justify-content: space-between; align-items: center; }
        .history-item:hover, .saved-item:hover { background: var(--hover); }
        .history-item .query { font-size: 0.875rem; color: var(--text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 180px; }
        .saved-item .name { font-size: 0.875rem; font-weight: 500; color: var(--text); }
        .saved-item .delete-btn { background: none; border: none; color: var(--text-muted); cursor: pointer; padding: 0.25rem; border-radius: 4px; font-size: 1rem; }
        .saved-item .delete-btn:hover { color: var(--error); }
        .modal-overlay { position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: rgba(0, 0, 0, 0.75); display: none; justify-content: center; align-items: center; z-index: 2000; padding: 2rem; backdrop-filter: blur(4px); }
        .modal-overlay.show { display: flex; }
        .modal { background: var(--card-bg); border-radius: 20px; width: 100%; max-width: 900px; max-height: 85vh; overflow: hidden; display: flex; flex-direction: column; border: 1px solid var(--border); box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5); }
        .modal-header { padding: 1.5rem 2rem; border-bottom: 1px solid var(--border); display: flex; justify-content: space-between; align-items: center; }
        .modal-header h3 { font-size: 1.25rem; font-weight: 600; }
        .modal-close { background: none; border: none; font-size: 1.5rem; cursor: pointer; color: var(--text-muted); width: 36px; height: 36px; border-radius: 8px; display: flex; align-items: center; justify-content: center; }
        .modal-close:hover { background: var(--hover); color: var(--text); }
        .modal-body { padding: 1.5rem 2rem; overflow-y: auto; flex: 1; }
        .preview-path { color: var(--text-secondary); font-size: 0.875rem; margin-bottom: 1rem; padding: 0.75rem; background: var(--bg); border-radius: 8px; word-break: break-all; }
        .preview-content { background: var(--bg); padding: 1.25rem; border-radius: 12px; white-space: pre-wrap; font-family: 'JetBrains Mono', monospace; font-size: 0.85rem; line-height: 1.7; max-height: 500px; overflow-y: auto; color: var(--text-secondary); }
        .modal-actions { padding: 1rem 2rem; border-top: 1px solid var(--border); display: flex; gap: 0.75rem; justify-content: flex-end; }
        .main-layout { display: grid; grid-template-columns: 1fr 280px; gap: 1.5rem; }
        @media (max-width: 1024px) { .main-layout { grid-template-columns: 1fr; } .stats-bar { display: none; } }
        .index-input { width: 100%; padding: 1rem 1.25rem; font-size: 1rem; background: var(--bg); border: 2px solid var(--border); border-radius: 12px; color: var(--text); margin-bottom: 1rem; }
        .index-input:focus { outline: none; border-color: var(--primary); }
        .ingest-status { margin-top: 1rem; padding: 0.75rem 1rem; border-radius: 8px; font-size: 0.9rem; }
        .ingest-status.success { background: rgba(34, 197, 94, 0.1); color: var(--success); }
        .ingest-status.error { background: rgba(239, 68, 68, 0.1); color: var(--error); }
        .ingest-status.loading { background: rgba(99, 102, 241, 0.1); color: var(--primary-light); }
        .graph-container { background: var(--card-bg); border-radius: 16px; padding: 1.5rem; border: 1px solid var(--border); min-height: 400px; }
        .api-section { background: var(--card-bg); border-radius: 12px; padding: 1.5rem; border: 1px solid var(--border); margin-bottom: 1rem; }
        .api-section h3 { font-size: 1rem; margin-bottom: 1rem; color: var(--primary-light); }
        .api-section p { font-size: 0.9rem; color: var(--text-secondary); margin-bottom: 0.5rem; }
        .api-section code { background: var(--bg); padding: 0.2rem 0.5rem; border-radius: 4px; font-size: 0.85rem; color: var(--primary-light); }
    </style>
</head>
<body>
    <div class="app">
        <header class="header">
            <div class="logo"><div class="logo-icon">M</div><div><h1>Mindy</h1><p>Personal AI Memory & Knowledge Graph</p></div></div>
            <div class="stats-bar"><div class="stat-item">Docs: <span id="docCount">-</span></div><div class="stat-item">Chunks: <span id="chunkCount">-</span></div><div class="stat-item">Entities: <span id="entityCount">-</span></div></div>
        </header>
        <div class="search-container">
            <form class="search-form" id="searchForm">
                <div class="search-input-wrapper">
                    <input type="text" class="search-input" id="searchInput" placeholder="Ask anything... (try: search, index, export, api)">
                    <div class="autocomplete-dropdown" id="autocomplete"></div>
                </div>
                <button type="submit" class="btn btn-primary">Search</button>
                <button type="button" class="btn btn-secondary" id="saveSearchBtn">Save</button>
            </form>
        </div>
        <div class="tabs">
            <button class="tab active" data-tab="search">Search</button>
            <button class="tab" data-tab="index">Index Files</button>
            <button class="tab" data-tab="history">History</button>
            <button class="tab" data-tab="saved">Saved</button>
            <button class="tab" data-tab="graph">Knowledge Graph</button>
            <button class="tab" data-tab="api">API Docs</button>
        </div>
        <div class="main-layout">
            <div class="main-content">
                <div class="tab-content active" id="search">
                    <div class="results-container" id="resultsContainer"><div class="empty-state"><svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" /></svg><p>Start typing to search your knowledge base</p></div></div>
                </div>
                <div class="tab-content" id="index">
                    <div class="results-container"><h3 style="margin-bottom:1rem;">Index Files or Folders</h3><input type="text" class="index-input" id="ingestPath" placeholder="Enter path (e.g., C:\Users\You\Documents)"><button class="btn btn-primary" onclick="ingestPath()">Index Files</button><div id="ingestStatus"></div></div>
                </div>
                <div class="tab-content" id="graph"><div class="graph-container" id="graphContainer"><div class="loading"><div class="spinner"></div>Loading knowledge graph...</div></div></div>
                <div class="tab-content" id="api">
                    <div class="api-section"><h3>Search API</h3><p><code>GET /api/v1/search?q=query</code> - Semantic search</p><p><code>GET /api/v1/search/history</code> - Get history</p><p><code>GET /api/v1/search/saved</code> - Get saved</p></div>
                    <div class="api-section"><h3>Data API</h3><p><code>POST /api/v1/export</code> - Export to ZIP</p><p><code>POST /api/v1/import?path=file.zip</code> - Import</p><p><code>POST /api/v1/batch/delete?type=pdf</code> - Batch delete</p></div>
                    <div class="api-section"><h3>Graph API</h3><p><code>GET /api/v1/graph/traverse?start=entity:name&depth=2</code> - Traverse</p><p><code>GET /api/v1/graph/search?q=name&type=Entity</code> - Search</p></div>
                </div>
            </div>
            <div class="sidebar">
                <div class="panel" id="historyPanel"><h3>Recent Searches</h3><div id="historyList"></div></div>
                <div class="panel" id="savedPanel" style="display:none;"><h3>Saved Searches</h3><div id="savedList"></div></div>
            </div>
        </div>
    </div>
    <div class="modal-overlay" id="previewModal"><div class="modal"><div class="modal-header"><h3>Document Preview</h3><button class="modal-close" onclick="closeModal()">&times;</button></div><div class="modal-body"><div class="preview-path" id="previewPath"></div><div class="preview-content" id="previewContent"></div></div><div class="modal-actions"><button class="btn btn-secondary" onclick="closeModal()">Close</button></div></div></div>
    <div class="modal-overlay" id="saveModal"><div class="modal"><div class="modal-header"><h3>Save Search</h3><button class="modal-close" onclick="closeSaveModal()">&times;</button></div><div class="modal-body"><p style="margin-bottom:1rem;color:var(--text-secondary);">Query: <strong id="saveQueryText" style="color:var(--text);"></strong></p><input type="text" class="index-input" id="saveNameInput" placeholder="Enter a name"></div><div class="modal-actions"><button class="btn btn-secondary" onclick="closeSaveModal()">Cancel</button><button class="btn btn-primary" onclick="saveSearch()">Save</button></div></div></div>
    <script>
        const API_BASE = window.location.origin; let currentQuery = ''; let autocompleteTimeout = null;
        loadStats(); loadHistory(); loadSavedSearches(); loadGraph();
        async function loadStats() { try { const res = await fetch(API_BASE + '/api/v1/stats'); const data = await res.json(); if (data.embedder) document.getElementById('docCount').textContent = data.embedder.doc_count || 0; if (data.file_tracker) document.getElementById('chunkCount').textContent = data.file_tracker.tracked_files || 0; } catch (e) {} }
        async function ingestPath() { const path = document.getElementById('ingestPath').value.trim(); if (!path) { document.getElementById('ingestStatus').innerHTML = '<div class="ingest-status error">Please enter a path</div>'; return; } document.getElementById('ingestStatus').innerHTML = '<div class="ingest-status loading">Indexing...</div>'; try { const response = await fetch(API_BASE + '/api/v1/ingest?path=' + encodeURIComponent(path), { method: 'POST' }); const data = await response.json(); if (data.status === 'ok') { document.getElementById('ingestStatus').innerHTML = '<div class="ingest-status success">✓ Indexed ' + (data.files || 1) + ' file(s)!</div>'; loadStats(); } else { document.getElementById('ingestStatus').innerHTML = '<div class="ingest-status error">Error: ' + data.message + '</div>'; } } catch (e) { document.getElementById('ingestStatus').innerHTML = '<div class="ingest-status error">Error: ' + e.message + '</div>'; } }
        document.getElementById('ingestPath').addEventListener('keypress', function(e) { if (e.key === 'Enter') ingestPath(); });
        document.getElementById('saveNameInput').addEventListener('keypress', function(e) { if (e.key === 'Enter') saveSearch(); });
        document.querySelectorAll('.tab').forEach(tab => { tab.addEventListener('click', () => { document.querySelectorAll('.tab').forEach(t => t.classList.remove('active')); document.querySelectorAll('.tab-content').forEach(c => c.classList.remove('active')); tab.classList.add('active'); document.getElementById(tab.dataset.tab).classList.add('active'); if (tab.dataset.tab === 'history') { document.getElementById('historyPanel').style.display = 'block'; document.getElementById('savedPanel').style.display = 'none'; } else if (tab.dataset.tab === 'saved') { document.getElementById('historyPanel').style.display = 'none'; document.getElementById('savedPanel').style.display = 'block'; } }); });
        document.getElementById('searchForm').addEventListener('submit', async (e) => { e.preventDefault(); const query = document.getElementById('searchInput').value; if (!query) return; currentQuery = query; performSearch(query); });
        document.getElementById('saveSearchBtn').addEventListener('click', () => { const query = document.getElementById('searchInput').value; if (!query) return; document.getElementById('saveQueryText').textContent = query; document.getElementById('saveNameInput').value = ''; document.getElementById('saveModal').classList.add('show'); });
        async function performSearch(query) { const container = document.getElementById('resultsContainer'); container.innerHTML = '<div class="loading"><div class="spinner"></div>Searching...</div>'; try { const response = await fetch(API_BASE + '/api/v1/search?q=' + encodeURIComponent(query) + '&k=20'); const data = await response.json(); if (data.results && data.results.length > 0) { container.innerHTML = '<div class="results-header"><h3>' + data.results.length + ' results for "' + query + '"</h3></div>' + data.results.map((r, i) => { let meta = {}; try { meta = JSON.parse(r.meta); } catch (e) {} const path = meta.path || r.id; return '<div class="result-item" onclick="showPreview(\'' + r.id + '\', \'' + encodeURIComponent(path) + '\')">' + '<span class="result-score">' + (r.score * 100).toFixed(1) + '%</span>' + '<span class="result-path">' + path + '</span>' + '</div>'; }).join(''); } else { container.innerHTML = '<div class="empty-state"><svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9.172 16.172a4 4 0 015.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /></svg><p>No results found for "' + query + '"</p></div>'; } } catch (e) { container.innerHTML = '<div class="empty-state"><p style="color:var(--error);">Error: ' + e.message + '</p></div>'; } }
        document.getElementById('searchInput').addEventListener('input', async (e) => { const query = e.target.value.trim(); if (query.length < 2) { document.getElementById('autocomplete').classList.remove('show'); return; } clearTimeout(autocompleteTimeout); autocompleteTimeout = setTimeout(async () => { try { const [historyRes, savedRes] = await Promise.all([fetch(API_BASE + '/api/v1/search/history?limit=5'), fetch(API_BASE + '/api/v1/search/saved')]); const history = await historyRes.json(); const saved = await savedRes.json(); let items = []; history.history.forEach(h => { if (h.query.toLowerCase().includes(query.toLowerCase())) items.push({ type: 'history', label: h.query, sub: 'Recent' }); }); saved.saved.forEach(s => { if (s.name.toLowerCase().includes(query.toLowerCase()) || s.query.toLowerCase().includes(query.toLowerCase())) items.push({ type: 'saved', label: s.name, sub: s.query }); }); items = items.slice(0, 8); if (items.length > 0) { const html = items.map(item => '<div class="autocomplete-item" onclick="selectAutocomplete(\'' + encodeURIComponent(item.label) + '\')">' + '<span class="label">' + item.label + '</span>' + '<span class="type">' + item.sub + '</span>' + '</div>').join(''); document.getElementById('autocomplete').innerHTML = html; document.getElementById('autocomplete').classList.add('show'); } else { document.getElementById('autocomplete').classList.remove('show'); } } catch (e) {} }, 150); });
        function selectAutocomplete(label) { document.getElementById('searchInput').value = decodeURIComponent(label); document.getElementById('autocomplete').classList.remove('show'); performSearch(decodeURIComponent(label)); }
        document.addEventListener('click', (e) => { if (!e.target.closest('.search-input-wrapper')) document.getElementById('autocomplete').classList.remove('show'); });
        async function loadHistory() { try { const response = await fetch(API_BASE + '/api/v1/search/history?limit=10'); const data = await response.json(); if (data.history && data.history.length > 0) { const html = data.history.map(h => '<div class="history-item" onclick="performSearch(\'' + encodeURIComponent(h.query) + '\'); document.getElementById(\'searchInput\').value=\'' + h.query + '\'">' + '<span class="query">' + h.query + '</span>' + '</div>').join(''); document.getElementById('historyList').innerHTML = html; } else { document.getElementById('historyList').innerHTML = '<p style="color:var(--text-muted);font-size:0.8rem;">No recent searches</p>'; } } catch (e) { document.getElementById('historyList').innerHTML = '<p style="color:var(--text-muted);font-size:0.8rem;">Error loading</p>'; } }
        async function loadSavedSearches() { try { const response = await fetch(API_BASE + '/api/v1/search/saved'); const data = await response.json(); if (data.saved && data.saved.length > 0) { const html = data.saved.map(s => '<div class="saved-item">' + '<span class="name" onclick="performSearch(\'' + encodeURIComponent(s.query) + '\'); document.getElementById(\'searchInput\').value=\'' + s.name + '\'">' + s.name + '</span>' + '<button class="delete-btn" onclick="event.stopPropagation();deleteSavedSearch(\'' + s.id + '\')">&times;</button>' + '</div>').join(''); document.getElementById('savedList').innerHTML = html; } else { document.getElementById('savedList').innerHTML = '<p style="color:var(--text-muted);font-size:0.8rem;">No saved searches</p>'; } } catch (e) { document.getElementById('savedList').innerHTML = '<p style="color:var(--text-muted);font-size:0.8rem;">Error loading</p>'; } }
        async function saveSearch() { const name = document.getElementById('saveNameInput').value.trim(); const query = document.getElementById('saveQueryText').textContent; if (!name) return; try { await fetch(API_BASE + '/api/v1/search/saved', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ name, query }) }); closeSaveModal(); loadSavedSearches(); } catch (e) { alert('Error saving: ' + e.message); } }
        async function deleteSavedSearch(id) { try { await fetch(API_BASE + '/api/v1/search/saved/' + id, { method: 'DELETE' }); loadSavedSearches(); } catch (e) {} }
        function closeModal() { document.getElementById('previewModal').classList.remove('show'); }
        function closeSaveModal() { document.getElementById('saveModal').classList.remove('show'); }
        async function showPreview(resultId, path) { document.getElementById('previewPath').textContent = decodeURIComponent(path); document.getElementById('previewContent').textContent = 'Loading...'; document.getElementById('previewModal').classList.add('show'); try { let docId = resultId; if (resultId.startsWith('chunk:')) { const parts = resultId.split(':'); if (parts.length >= 2) docId = 'doc:' + parts[1]; } const nodeRes = await fetch(API_BASE + '/api/v1/graph/node/' + docId); if (!nodeRes.ok) { const altDocId = resultId.replace('chunk:', 'doc:').split(':').slice(0,2).join(':'); const altRes = await fetch(API_BASE + '/api/v1/graph/node/' + altDocId); if (altRes.ok) { const node = await altRes.json(); if (node.blob_ref) { const blobRes = await fetch(API_BASE + '/api/v1/blob/' + node.blob_ref); const text = await blobRes.text(); document.getElementById('previewContent').textContent = text; return; } } document.getElementById('previewContent').textContent = 'Document not found'; return; } const node = await nodeRes.json(); if (node.blob_ref) { const blobRes = await fetch(API_BASE + '/api/v1/blob/' + node.blob_ref); const text = await blobRes.text(); document.getElementById('previewContent').textContent = text; } else { document.getElementById('previewContent').textContent = 'No content available'; } } catch (e) { document.getElementById('previewContent').textContent = 'Error: ' + e.message; } }
        document.addEventListener('keydown', (e) => { if (e.key === 'Escape') { closeModal(); closeSaveModal(); } });
        async function loadGraph() { const container = document.getElementById('graphContainer'); try { const response = await fetch(API_BASE + '/api/v1/graph/search?type=Entity&limit=30'); const data = await response.json(); if (data.nodes && data.nodes.length > 0) { const nodes = data.nodes; container.innerHTML = '<div style="margin-bottom:1rem;"><h3 style="color:var(--text);margin-bottom:0.5rem;">Knowledge Graph - ' + nodes.length + ' Entities</h3><p style="color:var(--text-muted);font-size:0.85rem;">Click an entity to explore connections</p></div><div style="display:flex;flex-wrap:wrap;gap:0.5rem;">' + nodes.map(n => '<span style="background:var(--bg);padding:0.5rem 1rem;border-radius:20px;font-size:0.85rem;cursor:pointer;transition:all 0.2s;" onclick="traverseGraph(\'' + n.id + '\')">' + (n.label || n.id).substring(0, 20) + '</span>').join('') + '</div>'; } else { container.innerHTML = '<div class="empty-state"><p>No entities found. Index some documents first!</p></div>'; } } catch (e) { container.innerHTML = '<div class="empty-state"><p style="color:var(--error);">Error: ' + e.message + '</p></div>'; } }
        function traverseGraph(entityId) { fetch(API_BASE + '/api/v1/graph/traverse?start=' + entityId + '&depth=2').then(r => r.json()).then(data => { if (data.nodes && data.nodes.length > 0) alert('Found ' + data.nodes.length + ' connected nodes'); }); }
    </script>
</body>
</html>`
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; color: #333; line-height: 1.6; }
        .header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 1.5rem; text-align: center; }
        .header h1 { font-size: 2rem; margin-bottom: 0.25rem; }
        .header p { font-size: 0.9rem; opacity: 0.9; }
        .container { max-width: 1200px; margin: 0 auto; padding: 1.5rem; }
        .search-wrapper { position: relative; margin-bottom: 1.5rem; }
        .search-box { background: white; border-radius: 12px; padding: 1.5rem; box-shadow: 0 4px 6px rgba(0,0,0,0.1); }
        .search-box h2 { margin-bottom: 1rem; color: #667eea; font-size: 1.25rem; }
        .search-form { display: flex; gap: 0.75rem; flex-wrap: wrap; }
        .search-input-wrapper { position: relative; flex: 1; min-width: 250px; }
        .search-input { width: 100%; padding: 0.75rem 1rem; font-size: 1rem; border: 2px solid #e0e0e0; border-radius: 8px; }
        .search-input:focus { outline: none; border-color: #667eea; }
        .autocomplete-dropdown { position: absolute; top: 100%; left: 0; right: 0; background: white; border: 1px solid #e0e0e0; border-radius: 0 0 8px 8px; max-height: 300px; overflow-y: auto; z-index: 1000; box-shadow: 0 4px 6px rgba(0,0,0,0.1); display: none; }
        .autocomplete-dropdown.show { display: block; }
        .autocomplete-item { padding: 0.75rem 1rem; cursor: pointer; border-bottom: 1px solid #f0f0f0; }
        .autocomplete-item:hover { background: #f5f5f5; }
        .autocomplete-item .label { font-weight: 500; }
        .autocomplete-item .type { font-size: 0.75rem; color: #999; margin-left: 0.5rem; }
        .btn { padding: 0.75rem 1.25rem; font-size: 0.9rem; border: none; border-radius: 8px; cursor: pointer; }
        .btn-primary { background: #667eea; color: white; }
        .btn-primary:hover { background: #5568d3; }
        .btn-secondary { background: #e0e0e0; color: #333; }
        .btn-secondary:hover { background: #d0d0d0; }
        .btn-sm { padding: 0.4rem 0.75rem; font-size: 0.8rem; }
        .tabs { display: flex; gap: 0.5rem; margin-bottom: 1rem; border-bottom: 2px solid #e0e0e0; padding-bottom: 0.5rem; flex-wrap: wrap; }
        .tab { padding: 0.5rem 1rem; border: none; background: none; cursor: pointer; font-size: 0.9rem; color: #666; border-radius: 8px 8px 0 0; }
        .tab:hover { background: #f0f0f0; }
        .tab.active { color: #667eea; background: rgba(102, 126, 234, 0.1); }
        .tab-content { display: none; }
        .tab-content.active { display: block; }
        .results { background: white; border-radius: 12px; padding: 1.5rem; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .result-item { padding: 1rem; border-bottom: 1px solid #e0e0e0; cursor: pointer; }
        .result-item:hover { background: #f9f9f9; }
        .result-item:last-child { border-bottom: none; }
        .result-score { display: inline-block; background: #667eea; color: white; padding: 0.2rem 0.5rem; border-radius: 4px; font-size: 0.75rem; margin-right: 0.5rem; }
        .result-meta { color: #666; font-size: 0.85rem; }
        .result-path { font-weight: 500; color: #333; }
        .loading { text-align: center; padding: 2rem; color: #666; }
        
        /* History & Saved Searches */
        .sidebar { background: white; border-radius: 12px; padding: 1rem; box-shadow: 0 2px 4px rgba(0,0,0,0.1); margin-bottom: 1rem; }
        .sidebar h3 { font-size: 0.9rem; color: #666; margin-bottom: 0.75rem; }
        .history-item, .saved-item { padding: 0.5rem 0.75rem; cursor: pointer; border-radius: 6px; margin-bottom: 0.25rem; display: flex; justify-content: space-between; align-items: center; }
        .history-item:hover, .saved-item:hover { background: #f5f5f5; }
        .history-item .query { font-size: 0.85rem; }
        .history-item .time { font-size: 0.7rem; color: #999; }
        .saved-item .name { font-size: 0.85rem; font-weight: 500; }
        .saved-item .actions { display: flex; gap: 0.25rem; }
        
        /* Modal */
        .modal-overlay { position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: rgba(0,0,0,0.5); display: none; justify-content: center; align-items: center; z-index: 2000; }
        .modal-overlay.show { display: flex; }
        .modal { background: white; border-radius: 12px; width: 90%; max-width: 800px; max-height: 80vh; overflow: hidden; display: flex; flex-direction: column; }
        .modal-header { padding: 1rem 1.5rem; border-bottom: 1px solid #e0e0e0; display: flex; justify-content: space-between; align-items: center; }
        .modal-header h3 { font-size: 1.1rem; color: #333; }
        .modal-close { background: none; border: none; font-size: 1.5rem; cursor: pointer; color: #999; }
        .modal-close:hover { color: #333; }
        .modal-body { padding: 1.5rem; overflow-y: auto; flex: 1; }
        .modal-actions { padding: 1rem 1.5rem; border-top: 1px solid #e0e0e0; display: flex; gap: 0.5rem; justify-content: flex-end; }
        
        /* Document Preview */
        .preview-path { color: #666; font-size: 0.85rem; margin-bottom: 1rem; }
        .preview-content { background: #f9f9f9; padding: 1rem; border-radius: 8px; white-space: pre-wrap; font-family: 'Consolas', 'Monaco', monospace; font-size: 0.85rem; max-height: 400px; overflow-y: auto; line-height: 1.5; }
        
        /* Graph Visualization */
        .graph-container { background: white; border-radius: 12px; padding: 1.5rem; box-shadow: 0 2px 4px rgba(0,0,0,0.1); height: 500px; position: relative; }
        .graph-svg { width: 100%; height: 100%; }
        .graph-node { cursor: pointer; }
        .graph-node circle { fill: #667eea; stroke: #fff; stroke-width: 2; }
        .graph-node text { font-size: 10px; fill: #333; }
        .graph-edge { stroke: #ccc; stroke-width: 1; }
        
        /* API Endpoints */
        .api-endpoints { background: white; border-radius: 12px; padding: 1.5rem; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .api-endpoints h3 { margin-bottom: 1rem; color: #333; }
        .api-endpoints p { margin-bottom: 0.5rem; font-size: 0.9rem; }
        
        /* Layout */
        .main-layout { display: grid; grid-template-columns: 1fr 300px; gap: 1.5rem; }
        @media (max-width: 900px) { .main-layout { grid-template-columns: 1fr; } }
        
        /* Empty State */
        .empty-state { text-align: center; padding: 2rem; color: #999; }
    </style>
</head>
<body>
    <div class="header">
        <h1>Mindy</h1>
        <p>Personal AI Memory & Knowledge Graph</p>
    </div>
    <div class="container">
        <div class="search-wrapper">
            <div class="search-box">
                <h2>Search</h2>
                <form class="search-form" id="searchForm">
                    <div class="search-input-wrapper">
                        <input type="text" class="search-input" id="searchInput" placeholder="Ask anything..." autocomplete="off">
                        <div class="autocomplete-dropdown" id="autocomplete"></div>
                    </div>
                    <button type="submit" class="btn btn-primary">Search</button>
                    <button type="button" class="btn btn-secondary" id="saveSearchBtn">Save</button>
                </form>
            </div>
        </div>
        
        <div class="tabs">
            <button class="tab active" data-tab="search">Search</button>
            <button class="tab" data-tab="index">Index Files</button>
            <button class="tab" data-tab="history">History</button>
            <button class="tab" data-tab="saved">Saved</button>
            <button class="tab" data-tab="graph">Graph</button>
            <button class="tab" data-tab="api">API</button>
        </div>
        
        <div class="tab-content active" id="search">
            <div class="results" id="resultsContainer">
                <p class="empty-state">Enter a search query above</p>
            </div>
        </div>
        
        <div class="tab-content" id="index">
            <div class="results">
                <h3 style="margin-bottom:1rem;">Index Files or Folders</h3>
                <div style="margin-bottom:1rem;">
                    <input type="text" class="search-input" id="ingestPath" placeholder="Enter path (e.g., C:\Users\You\Documents)" style="width:100%;">
                </div>
                <button class="btn btn-primary" onclick="ingestPath()">Index</button>
                <div id="ingestStatus" style="margin-top:1rem;"></div>
            </div>
        </div>
        
        <div class="main-layout">
            <div class="main-content">
                <div class="tab-content active" id="search">
                    <div class="results" id="resultsContainer">
                        <p class="empty-state">Enter a search query above</p>
                    </div>
                </div>
                
                <div class="tab-content" id="graph">
                    <div class="graph-container">
                        <svg class="graph-svg" id="graphSvg">
                            <text x="50%" y="50%" text-anchor="middle" fill="#999">Loading graph...</text>
                        </svg>
                    </div>
                </div>
                
                <div class="tab-content" id="api">
                    <div class="api-endpoints">
                        <h3>Search API</h3>
                        <p><strong>GET /api/v1/search?q=&lt;query&gt;</strong> - Semantic search</p>
                        <p><strong>GET /api/v1/search/history</strong> - Get search history</p>
                        <p><strong>GET /api/v1/search/saved</strong> - Get saved searches</p>
                        <h3>Data API</h3>
                        <p><strong>POST /api/v1/export</strong> - Export data (?output=&lt;path&gt;)</p>
                        <p><strong>POST /api/v1/import?path=&lt;path&gt;</strong> - Import data</p>
                        <p><strong>POST /api/v1/batch/delete</strong> - Batch delete (?path=&lt;pattern&gt;)</p>
                        <h3>Graph API</h3>
                        <p><strong>GET /api/v1/graph/traverse?start=&lt;id&gt;&amp;depth=&lt;n&gt;</strong> - Traverse graph</p>
                        <p><strong>GET /api/v1/graph/search?q=&lt;query&gt;</strong> - Search nodes</p>
                    </div>
                </div>
            </div>
            
            <div class="sidebar-content">
                <div class="sidebar" id="historySidebar">
                    <h3>Recent Searches</h3>
                    <div id="historyList"></div>
                </div>
                
                <div class="sidebar" id="savedSidebar" style="display:none;">
                    <h3>Saved Searches</h3>
                    <div id="savedList"></div>
                </div>
            </div>
        </div>
    </div>
    
    <!-- Document Preview Modal -->
    <div class="modal-overlay" id="previewModal">
        <div class="modal">
            <div class="modal-header">
                <h3>Document Preview</h3>
                <button class="modal-close" onclick="closeModal()">&times;</button>
            </div>
            <div class="modal-body">
                <div class="preview-path" id="previewPath"></div>
                <div class="preview-content" id="previewContent"></div>
            </div>
            <div class="modal-actions">
                <button class="btn btn-secondary btn-sm" onclick="closeModal()">Close</button>
            </div>
        </div>
    </div>
    
    <!-- Save Search Modal -->
    <div class="modal-overlay" id="saveModal">
        <div class="modal">
            <div class="modal-header">
                <h3>Save Search</h3>
                <button class="modal-close" onclick="closeSaveModal()">&times;</button>
            </div>
            <div class="modal-body">
                <p style="margin-bottom: 1rem;">Query: <strong id="saveQueryText"></strong></p>
                <input type="text" class="search-input" id="saveNameInput" placeholder="Enter a name for this search">
            </div>
            <div class="modal-actions">
                <button class="btn btn-secondary btn-sm" onclick="closeSaveModal()">Cancel</button>
                <button class="btn btn-primary btn-sm" onclick="saveSearch()">Save</button>
            </div>
        </div>
    </div>

    <script>
        const API_BASE = window.location.origin;
        let currentQuery = '';
        let autocompleteTimeout = null;
        
        // Initialize
        loadHistory();
        loadSavedSearches();
        loadGraph();
        
        // Ingest function
        async function ingestPath() {
            const path = document.getElementById('ingestPath').value.trim();
            if (!path) {
                document.getElementById('ingestStatus').innerHTML = '<span style="color:red;">Please enter a path</span>';
                return;
            }
            
            document.getElementById('ingestStatus').innerHTML = '<span style="color:#666;">Indexing...</span>';
            
            try {
                const response = await fetch(API_BASE + '/api/v1/ingest?path=' + encodeURIComponent(path), {
                    method: 'POST'
                });
                const data = await response.json();
                
                if (data.status === 'ok') {
                    document.getElementById('ingestStatus').innerHTML = '<span style="color:green;">Indexed ' + (data.files || 1) + ' file(s)!</span>';
                } else {
                    document.getElementById('ingestStatus').innerHTML = '<span style="color:red;">Error: ' + data.message + '</span>';
                }
            } catch (e) {
                document.getElementById('ingestStatus').innerHTML = '<span style="color:red;">Error: ' + e.message + '</span>';
            }
        }
        
        // Enter key for ingest
        document.getElementById('ingestPath').addEventListener('keypress', function (e) {
            if (e.key === 'Enter') {
                ingestPath();
            }
        });
        
        // Tab switching
        document.querySelectorAll('.tab').forEach(tab => {
            tab.addEventListener('click', () => {
                document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
                document.querySelectorAll('.tab-content').forEach(c => c.classList.remove('active'));
                tab.classList.add('active');
                document.getElementById(tab.dataset.tab).classList.add('active');
                
                if (tab.dataset.tab === 'history') {
                    document.getElementById('historySidebar').style.display = 'block';
                    document.getElementById('savedSidebar').style.display = 'none';
                } else if (tab.dataset.tab === 'saved') {
                    document.getElementById('historySidebar').style.display = 'none';
                    document.getElementById('savedSidebar').style.display = 'block';
                }
            });
        });
        
        // Search form
        document.getElementById('searchForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            const query = document.getElementById('searchInput').value;
            if (!query) return;
            currentQuery = query;
            performSearch(query);
        });
        
        // Save search button
        document.getElementById('saveSearchBtn').addEventListener('click', () => {
            const query = document.getElementById('searchInput').value;
            if (!query) return;
            document.getElementById('saveQueryText').textContent = query;
            document.getElementById('saveNameInput').value = '';
            document.getElementById('saveModal').classList.add('show');
        });
        
        async function performSearch(query) {
            const container = document.getElementById('resultsContainer');
            container.innerHTML = '<div class="loading">Searching...</div>';
            
            try {
                const response = await fetch(API_BASE + '/api/v1/search?q=' + encodeURIComponent(query) + '&k=20');
                const data = await response.json();
                
                console.log('Search response:', data);
                
                if (data.results && data.results.length > 0) {
                    container.innerHTML = '<h3 style="margin-bottom:1rem;">Results (' + data.results.length + ')</h3>' + data.results.map((r, i) => {
                        let meta = {};
                        try { meta = JSON.parse(r.meta); } catch (e) {}
                        const path = meta.path || r.id;
                        return '<div class="result-item" onclick="showPreview(\'' + r.id + '\', \'' + encodeURIComponent(path) + '\')">' +
                            '<span class="result-score">' + (r.score * 100).toFixed(1) + '%</span> ' +
                            '<span class="result-path">' + path + '</span>' +
                            '</div>';
                    }).join('');
                } else {
                    container.innerHTML = '<p class="empty-state">No results found</p><pre style="text-align:left;background:#f5f5f5;padding:1rem;border-radius:8px;margin-top:1rem;">Debug: ' + JSON.stringify(data, null, 2) + '</pre>';
                }
            } catch (e) {
                container.innerHTML = '<p style="color: red;">Error: ' + e.message + '</p>';
            }
        }
        
        // Autocomplete
        document.getElementById('searchInput').addEventListener('input', async (e) => {
            const query = e.target.value.trim();
            if (query.length < 2) {
                document.getElementById('autocomplete').classList.remove('show');
                return;
            }
            
            clearTimeout(autocompleteTimeout);
            autocompleteTimeout = setTimeout(async () => {
                try {
                    const [historyRes, savedRes] = await Promise.all([
                        fetch(API_BASE + '/api/v1/search/history?limit=5'),
                        fetch(API_BASE + '/api/v1/search/saved')
                    ]);
                    
                    const history = await historyRes.json();
                    const saved = await savedRes.json();
                    
                    let items = [];
                    
                    // Add matching history
                    history.history.forEach(h => {
                        if (h.query.toLowerCase().includes(query.toLowerCase())) {
                            items.push({ type: 'history', label: h.query, sub: 'Recent' });
                        }
                    });
                    
                    // Add matching saved searches
                    saved.saved.forEach(s => {
                        if (s.name.toLowerCase().includes(query.toLowerCase()) || s.query.toLowerCase().includes(query.toLowerCase())) {
                            items.push({ type: 'saved', label: s.name, sub: s.query });
                        }
                    });
                    
                    // Limit to 8 items
                    items = items.slice(0, 8);
                    
                    if (items.length > 0) {
                        const html = items.map(item => 
                            '<div class="autocomplete-item" onclick="selectAutocomplete(\'' + encodeURIComponent(item.label) + '\')">' +
                            '<span class="label">' + item.label + '</span>' +
                            '<span class="type">' + item.sub + '</span>' +
                            '</div>'
                        ).join('');
                        document.getElementById('autocomplete').innerHTML = html;
                        document.getElementById('autocomplete').classList.add('show');
                    } else {
                        document.getElementById('autocomplete').classList.remove('show');
                    }
                } catch (e) {
                    console.error('Autocomplete error:', e);
                }
            }, 200);
        });
        
        function selectAutocomplete(label) {
            document.getElementById('searchInput').value = decodeURIComponent(label);
            document.getElementById('autocomplete').classList.remove('show');
            performSearch(decodeURIComponent(label));
        }
        
        // Close autocomplete when clicking outside
        document.addEventListener('click', (e) => {
            if (!e.target.closest('.search-input-wrapper')) {
                document.getElementById('autocomplete').classList.remove('show');
            }
        });
        
        // Load history
        async function loadHistory() {
            try {
                const response = await fetch(API_BASE + '/api/v1/search/history?limit=10');
                const data = await response.json();
                
                if (data.history && data.history.length > 0) {
                    const html = data.history.map(h => 
                        '<div class="history-item" onclick="performSearch(\'' + encodeURIComponent(h.query) + '\')">' +
                        '<span class="query">' + h.query + '</span>' +
                        '</div>'
                    ).join('');
                    document.getElementById('historyList').innerHTML = html;
                } else {
                    document.getElementById('historyList').innerHTML = '<p class="empty-state" style="font-size:0.8rem;">No recent searches</p>';
                }
            } catch (e) {
                document.getElementById('historyList').innerHTML = '<p class="empty-state" style="font-size:0.8rem;">Error loading history</p>';
            }
        }
        
        // Load saved searches
        async function loadSavedSearches() {
            try {
                const response = await fetch(API_BASE + '/api/v1/search/saved');
                const data = await response.json();
                
                if (data.saved && data.saved.length > 0) {
                    const html = data.saved.map(s => 
                        '<div class="saved-item">' +
                        '<span class="name" onclick="performSearch(\'' + encodeURIComponent(s.query) + '\')">' + s.name + '</span>' +
                        '<div class="actions">' +
                        '<button class="btn btn-secondary btn-sm" onclick="deleteSavedSearch(\'' + s.id + '\')">×</button>' +
                        '</div>' +
                        '</div>'
                    ).join('');
                    document.getElementById('savedList').innerHTML = html;
                } else {
                    document.getElementById('savedList').innerHTML = '<p class="empty-state" style="font-size:0.8rem;">No saved searches</p>';
                }
            } catch (e) {
                document.getElementById('savedList').innerHTML = '<p class="empty-state" style="font-size:0.8rem;">Error loading saved</p>';
            }
        }
        
        // Save search
        async function saveSearch() {
            const name = document.getElementById('saveNameInput').value.trim();
            const query = document.getElementById('saveQueryText').textContent;
            
            if (!name) return;
            
            try {
                await fetch(API_BASE + '/api/v1/search/saved', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ name, query })
                });
                closeSaveModal();
                loadSavedSearches();
            } catch (e) {
                alert('Error saving: ' + e.message);
            }
        }
        
        // Delete saved search
        async function deleteSavedSearch(id) {
            try {
                await fetch(API_BASE + '/api/v1/search/saved/' + id, { method: 'DELETE' });
                loadSavedSearches();
            } catch (e) {
                alert('Error deleting: ' + e.message);
            }
        }
        
        // Modal functions
        function closeModal() {
            document.getElementById('previewModal').classList.remove('show');
        }
        
        function closeSaveModal() {
            document.getElementById('saveModal').classList.remove('show');
        }
        
        // Show document preview
        async function showPreview(resultId, path) {
            document.getElementById('previewPath').textContent = decodeURIComponent(path);
            document.getElementById('previewContent').textContent = 'Loading...';
            document.getElementById('previewModal').classList.add('show');
            
            try {
                // Try to find doc_id from the search result
                // The result ID format is "chunk:docHash:chunkIndex:chunkHash"
                let docId = resultId;
                if (resultId.startsWith('chunk:')) {
                    const parts = resultId.split(':');
                    if (parts.length >= 2) {
                        docId = 'doc:' + parts[1];
                    }
                }
                
                // Get the document node to find blob ref
                const nodeRes = await fetch(API_BASE + '/api/v1/graph/node/' + docId);
                
                if (!nodeRes.ok) {
                    // Try alternate format
                    const altDocId = resultId.replace('chunk:', 'doc:').split(':').slice(0,2).join(':');
                    const altRes = await fetch(API_BASE + '/api/v1/graph/node/' + altDocId);
                    if (altRes.ok) {
                        const node = await altRes.json();
                        if (node.blob_ref) {
                            const blobRes = await fetch(API_BASE + '/api/v1/blob/' + node.blob_ref);
                            const text = await blobRes.text();
                            document.getElementById('previewContent').textContent = text;
                            return;
                        }
                    }
                    document.getElementById('previewContent').textContent = 'Document not found. ID: ' + docId;
                    return;
                }
                
                const node = await nodeRes.json();
                
                if (node.blob_ref) {
                    const blobRes = await fetch(API_BASE + '/api/v1/blob/' + node.blob_ref);
                    const text = await blobRes.text();
                    document.getElementById('previewContent').textContent = text;
                } else {
                    document.getElementById('previewContent').textContent = 'No content available';
                }
            } catch (e) {
                document.getElementById('previewContent').textContent = 'Error loading: ' + e.message;
            }
        }
        
        // Close modal on escape
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                closeModal();
                closeSaveModal();
            }
        });
        
        // Load graph
        async function loadGraph() {
            const svg = document.getElementById('graphSvg');
            try {
                // Get some entities - using type filter without query
                const response = await fetch(API_BASE + '/api/v1/graph/search?type=Entity&limit=20');
                const data = await response.json();
                
                if (data.nodes && data.nodes.length > 0) {
                    const nodes = data.nodes;
                    const width = 800;
                    const height = 450;
                    
                    // Simple force-directed layout
                    const positions = {};
                    nodes.forEach((n, i) => {
                        const angle = (2 * Math.PI * i) / nodes.length;
                        positions[n.id] = {
                            x: width/2 + 150 * Math.cos(angle),
                            y: height/2 + 150 * Math.sin(angle)
                        };
                    });
                    
                    let html = '';
                    
                    // Draw edges (simplified - just connecting entities that might be related)
                    nodes.forEach((n, i) => {
                        if (i > 0) {
                            html += '<line class="graph-edge" x1="' + positions[nodes[0].id].x + '" y1="' + positions[nodes[0].id].y + 
                                    '" x2="' + positions[n.id].x + '" y2="' + positions[n.id].y + '"/>';
                        }
                    });
                    
                    // Draw nodes
                    nodes.forEach(n => {
                        html += '<g class="graph-node" onclick="traverseGraph(\'' + n.id + '\')">' +
                            '<circle cx="' + positions[n.id].x + '" cy="' + positions[n.id].y + '" r="15"/>' +
                            '<text x="' + positions[n.id].x + '" y="' + (positions[n.id].y + 30) + '" text-anchor="middle">' + 
                            (n.label ? n.label.substring(0, 10) : n.id.substring(0, 10)) + '</text></g>';
                    });
                    
                    svg.innerHTML = html;
                } else {
                    svg.innerHTML = '<text x="50%" y="50%" text-anchor="middle" fill="#999">No entities found. Index some documents first!</text>';
                }
            } catch (e) {
                svg.innerHTML = '<text x="50%" y="50%" text-anchor="middle" fill="#999">Error loading graph: ' + e.message + '</text>';
            }
        }
        
        function traverseGraph(entityId) {
            fetch(API_BASE + '/api/v1/graph/traverse?start=' + entityId + '&depth=2')
                .then(r => r.json())
                .then(data => {
                    if (data.nodes && data.nodes.length > 0) {
                        alert('Found ' + data.nodes.length + ' connected nodes');
                    }
                });
        }
    </script>
</body>
</html>`

type Server struct {
	port          int
	blobStore     *blob.Store
	vectorIndex   *vector.Index
	graphStore    *graph.Store
	indexer       *indexer.Indexer
	embedder      embedder.Embedder
	dataManager   *dataman.DataManager
	searchHistory *dataman.SearchHistory
	savedSearches *dataman.SavedSearches
	httpServer    *http.Server
}

func NewServer(port int, blobStore *blob.Store, vectorIndex *vector.Index, graphStore *graph.Store, idx *indexer.Indexer, dataDir string) *Server {
	var tfidf *embedder.TFIDF
	if idx != nil {
		tfidf = idx.GetEmbedder()
	}

	dm := dataman.NewDataManager(dataDir)
	sh := dataman.NewSearchHistory(dataDir, 100)
	sh.Load()
	ss := dataman.NewSavedSearches(dataDir)
	ss.Load()

	return &Server{
		port:          port,
		blobStore:     blobStore,
		vectorIndex:   vectorIndex,
		graphStore:    graphStore,
		indexer:       idx,
		embedder:      tfidf,
		dataManager:   dm,
		searchHistory: sh,
		savedSearches: ss,
	}
}

func (s *Server) Start() error {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Get("/", s.serveWebUI)
	r.Get("/ui", s.serveWebUI)
	r.Get("/health", s.health)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/ingest", s.ingest)
		r.Post("/reindex", s.reindex)
		r.Get("/search", s.search)
		r.Get("/stats", s.stats)
		r.Get("/graph/node/{id}", s.getNode)
		r.Get("/graph/traverse", s.traverse)
		r.Get("/graph/search", s.searchNodes)
		r.Get("/blob/{hash}", s.getBlob)

		// Export/Import
		r.Post("/export", s.exportData)
		r.Post("/import", s.importData)
		r.Post("/reset", s.resetData)

		// Batch operations
		r.Post("/batch/delete", s.batchDelete)
		r.Post("/batch/reindex", s.batchReindex)

		// Search history
		r.Get("/search/history", s.getSearchHistory)
		r.Delete("/search/history", s.clearSearchHistory)

		// Saved searches
		r.Get("/search/saved", s.getSavedSearches)
		r.Post("/search/saved", s.saveSearch)
		r.Put("/search/saved/{id}", s.updateSavedSearch)
		r.Delete("/search/saved/{id}", s.deleteSavedSearch)
	})

	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: r,
	}

	return s.httpServer.ListenAndServe()
}

func (s *Server) Stop() {
	if s.httpServer != nil {
		s.httpServer.Close()
	}
}

func (s *Server) serveWebUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(webUIHTML))
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func (s *Server) ingest(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "path required", http.StatusBadRequest)
		return
	}

	info, err := os.Stat(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if info.IsDir() {
		var indexed int
		filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !info.IsDir() {
				if s.indexer != nil {
					go s.indexer.IndexFile(p)
				}
				indexed++
			}
			return nil
		})
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "ok",
			"message": "Directory queued for indexing",
			"files":   indexed,
		})
		return
	}

	if s.indexer != nil {
		if err := s.indexer.IndexFile(path); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"path":   path,
	})
}

func (s *Server) reindex(w http.ResponseWriter, r *http.Request) {
	if s.indexer != nil {
		go s.indexer.ReindexAll()
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"message": "Reindex started in background",
		})
		return
	}
	
	http.Error(w, "indexer not available", http.StatusServiceUnavailable)
}

func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "q (query) required", http.StatusBadRequest)
		return
	}

	if s.embedder == nil || s.vectorIndex == nil {
		http.Error(w, "indexer not available", http.StatusServiceUnavailable)
		return
	}

	k := 10
	if kStr := r.URL.Query().Get("k"); kStr != "" {
		if parsed, err := strconv.Atoi(kStr); err == nil && parsed > 0 && parsed <= 100 {
			k = parsed
		}
	}

	offset := 0
	if offStr := r.URL.Query().Get("offset"); offStr != "" {
		if parsed, err := strconv.Atoi(offStr); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	fileType := r.URL.Query().Get("type")
	pathFilter := r.URL.Query().Get("path")

	queryVec, err := s.embedder.Embed(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	allResults, err := s.vectorIndex.Search(queryVec, k+offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var filteredResults []SearchResult
	
	for _, result := range allResults {
		if len(filteredResults) >= k {
			break
		}
		
		if fileType != "" {
			if !strings.Contains(result.Meta, `"file_type":"`+fileType) && 
			   !strings.Contains(result.Meta, `"content_type":"`+fileType) {
				continue
			}
		}
		
		if pathFilter != "" && !strings.Contains(result.Meta, pathFilter) {
			continue
		}
		
		filteredResults = append(filteredResults, SearchResult{
			ID:    result.ID,
			Score: result.Score,
			Meta:  result.Meta,
		})
	}

	if offset > len(filteredResults) {
		filteredResults = []SearchResult{}
	} else if offset > 0 && offset < len(filteredResults) {
		filteredResults = filteredResults[offset:]
	}

	response := SearchResponse{
		Query:      query,
		Results:    filteredResults,
		Total:     len(allResults),
		Offset:    offset,
		Limit:     k,
		Page:      offset/k + 1,
	}
	
	if len(allResults) > offset+k {
		response.NextOffset = offset + k
	}

	// Track search history
	if s.searchHistory != nil && query != "" {
		s.searchHistory.Add(query, len(filteredResults))
	}

	json.NewEncoder(w).Encode(response)
}

type SearchResponse struct {
	Query      string          `json:"query"`
	Results    []SearchResult  `json:"results"`
	Total      int             `json:"total"`
	Offset     int             `json:"offset"`
	Limit      int             `json:"limit"`
	Page       int             `json:"page"`
	NextOffset int             `json:"next_offset,omitempty"`
}

type SearchResult struct {
	ID    string  `json:"id"`
	Score float32 `json:"score"`
	Meta  string  `json:"meta"`
}

func (s *Server) stats(w http.ResponseWriter, r *http.Request) {
	stats := make(map[string]interface{})
	
	if s.indexer != nil {
		stats = s.indexer.GetStats()
	}
	
	stats["indexer"] = map[string]interface{}{
		"files_indexed": s.indexer.GetFileCount(),
	}
	
	json.NewEncoder(w).Encode(stats)
}

func (s *Server) searchNodes(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	nodeType := r.URL.Query().Get("type")

	limit := 20
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if parsed, err := strconv.Atoi(lStr); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	typeFilter := map[string]string{
		"document": "Document",
		"chunk":    "Chunk",
		"entity":   "Entity",
	}
	
	if nodeType != "" {
		if t, ok := typeFilter[nodeType]; ok {
			nodeType = t
		}
	}

	nodes := s.graphStore.SearchNodes(nodeType, query, limit)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"query": query,
		"nodes": nodes,
		"count": len(nodes),
	})
}

func (s *Server) getNode(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}

	node, err := s.graphStore.GetNode(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(node)
}

func (s *Server) traverse(w http.ResponseWriter, r *http.Request) {
	start := r.URL.Query().Get("start")
	if start == "" {
		http.Error(w, "start required", http.StatusBadRequest)
		return
	}

	edgeType := r.URL.Query().Get("type")
	depth := 3
	if dStr := r.URL.Query().Get("depth"); dStr != "" {
		if parsed, err := strconv.Atoi(dStr); err == nil && parsed > 0 && parsed <= 10 {
			depth = parsed
		}
	}

	nodes, err := s.graphStore.Traverse(start, edgeType, depth)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"start": start,
		"nodes": nodes,
		"count": len(nodes),
	})
}

func (s *Server) getBlob(w http.ResponseWriter, r *http.Request) {
	hash := chi.URLParam(r, "hash")
	if hash == "" {
		http.Error(w, "hash required", http.StatusBadRequest)
		return
	}

	data, err := s.blobStore.Get(hash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write(data)
}

// Export/Import handlers

func (s *Server) exportData(w http.ResponseWriter, r *http.Request) {
	outputPath := r.URL.Query().Get("output")
	
	opts := &dataman.ExportOptions{
		IncludeBlobs:   r.URL.Query().Get("blobs") != "false",
		IncludeGraph:   r.URL.Query().Get("graph") != "false",
		IncludeTFIDF:   r.URL.Query().Get("tfidf") != "false",
		IncludeHistory: r.URL.Query().Get("history") != "false",
		OutputPath:     outputPath,
	}

	if err := s.dataManager.Export(opts); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "ok",
		"output":     opts.OutputPath,
		"message":    "Export completed successfully",
	})
}

func (s *Server) importData(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "path required", http.StatusBadRequest)
		return
	}

	merge := r.URL.Query().Get("merge") == "true"

	opts := &dataman.ImportOptions{
		ImportPath: path,
		Merge:      merge,
	}

	if err := s.dataManager.Import(opts); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"message": "Import completed successfully",
	})
}

func (s *Server) resetData(w http.ResponseWriter, r *http.Request) {
	confirm := r.URL.Query().Get("confirm")
	if confirm != "yes" {
		http.Error(w, "confirm=yes required", http.StatusBadRequest)
		return
	}

	if err := s.dataManager.Reset(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"message": "All data has been reset",
	})
}

// Batch operations handlers

func (s *Server) batchDelete(w http.ResponseWriter, r *http.Request) {
	pathPattern := r.URL.Query().Get("path")
	fileType := r.URL.Query().Get("type")
	olderThanDays := 0
	if dStr := r.URL.Query().Get("older_than"); dStr != "" {
		if parsed, err := strconv.Atoi(dStr); err == nil {
			olderThanDays = parsed
		}
	}
	dryRun := r.URL.Query().Get("dry_run") == "true"

	opts := &dataman.BatchDeleteOptions{
		PathPattern:   pathPattern,
		FileType:      fileType,
		OlderThanDays: olderThanDays,
		DryRun:        dryRun,
	}

	count, err := s.dataManager.BatchDelete(opts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "ok",
		"deleted":    count,
		"dry_run":    dryRun,
		"message":    fmt.Sprintf("Deleted %d files", count),
	})
}

func (s *Server) batchReindex(w http.ResponseWriter, r *http.Request) {
	pathPattern := r.URL.Query().Get("path")
	fileType := r.URL.Query().Get("type")

	opts := &dataman.BatchReindexOptions{
		PathPattern: pathPattern,
		FileType:    fileType,
	}

	files, err := s.dataManager.BatchReindex(opts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, path := range files {
		if s.indexer != nil {
			go s.indexer.IndexFile(path)
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"files":   len(files),
		"message": fmt.Sprintf("Reindexing %d files", len(files)),
	})
}

// Search history handlers

func (s *Server) getSearchHistory(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if parsed, err := strconv.Atoi(lStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	history := s.searchHistory.GetRecent(limit)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"history": history,
		"count":   len(history),
	})
}

func (s *Server) clearSearchHistory(w http.ResponseWriter, r *http.Request) {
	s.searchHistory.Clear()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"message": "Search history cleared",
	})
}

// Saved searches handlers

func (s *Server) getSavedSearches(w http.ResponseWriter, r *http.Request) {
	saved := s.savedSearches.GetAll()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"saved": saved,
		"count": len(saved),
	})
}

func (s *Server) saveSearch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name  string `json:"name"`
		Query string `json:"query"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.Query == "" {
		http.Error(w, "name and query required", http.StatusBadRequest)
		return
	}

	search, err := s.savedSearches.Add(req.Name, req.Query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"search": search,
	})
}

func (s *Server) updateSavedSearch(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}

	var req struct {
		Name  string `json:"name"`
		Query string `json:"query"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	search, err := s.savedSearches.Update(id, req.Name, req.Query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if search == nil {
		http.Error(w, "search not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"search": search,
	})
}

func (s *Server) deleteSavedSearch(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}

	if err := s.savedSearches.Delete(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"message": "Search deleted",
	})
}
