// PROF Analyzer - Popup Script

let selectedFiles = [];
let analysisResult = null;

// DOM Elements
const uploadZone = document.getElementById('uploadZone');
const fileInput = document.getElementById('fileInput');
const fileList = document.getElementById('fileList');
const analyzeBtn = document.getElementById('analyzeBtn');
const resultsDiv = document.getElementById('results');
const resultsContent = document.getElementById('resultsContent');
const errorDiv = document.getElementById('error');
const apiUrlInput = document.getElementById('apiUrl');
const apiKeyInput = document.getElementById('apiKey');

// Load saved config
chrome.storage.local.get(['apiUrl', 'apiKey'], (result) => {
  if (result.apiUrl) {
    apiUrlInput.value = result.apiUrl;
  }
  if (result.apiKey) {
    apiKeyInput.value = result.apiKey;
  }
});

// Save config on change
apiUrlInput.addEventListener('change', () => {
  chrome.storage.local.set({ apiUrl: apiUrlInput.value });
});

apiKeyInput.addEventListener('change', () => {
  chrome.storage.local.set({ apiKey: apiKeyInput.value });
});

// File Upload Handling
uploadZone.addEventListener('click', () => fileInput.click());
uploadZone.addEventListener('dragover', (e) => {
  e.preventDefault();
  uploadZone.classList.add('dragover');
});
uploadZone.addEventListener('dragleave', () => {
  uploadZone.classList.remove('dragover');
});
uploadZone.addEventListener('drop', (e) => {
  e.preventDefault();
  uploadZone.classList.remove('dragover');
  handleFiles(e.dataTransfer.files);
});
fileInput.addEventListener('change', () => {
  handleFiles(fileInput.files);
});

function handleFiles(files) {
  for (const file of files) {
    selectedFiles.push(file);
  }
  renderFileList();
}

function renderFileList() {
  fileList.innerHTML = selectedFiles.map((file, index) => `
    <div class="file-item">
      <span class="name">${file.name}</span>
      <span class="size">${formatFileSize(file.size)}</span>
      <span class="remove" data-index="${index}">✕</span>
    </div>
  `).join('');
  
  // Add remove handlers
  document.querySelectorAll('.remove').forEach(btn => {
    btn.addEventListener('click', (e) => {
      const index = parseInt(e.target.dataset.index);
      selectedFiles.splice(index, 1);
      renderFileList();
    });
  });
}

function formatFileSize(bytes) {
  if (bytes < 1024) return bytes + ' B';
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
}

// Analyze Button
analyzeBtn.addEventListener('click', async () => {
  if (selectedFiles.length === 0) {
    showError('Please select at least one file');
    return;
  }
  
  const apiUrl = apiUrlInput.value.trim();
  if (!apiUrl) {
    showError('Please enter API URL');
    return;
  }
  
  const apiKey = apiKeyInput.value.trim();
  
  // Save config
  chrome.storage.local.set({ apiUrl, apiKey });
  
  // Show loading
  analyzeBtn.disabled = true;
  analyzeBtn.textContent = 'Analyzing...';
  errorDiv.style.display = 'none';
  resultsDiv.style.display = 'none';
  
  try {
    const formData = new FormData();
    selectedFiles.forEach(file => {
      formData.append('files', file);
    });
    
    const headers = {};
    if (apiKey) {
      headers['X-API-Key'] = apiKey;
    }
    
    const response = await fetch(`${apiUrl}/api/analyze/stream`, {
      method: 'POST',
      headers,
      body: formData
    });
    
    if (!response.ok) {
      throw new Error(`API error: ${response.status} ${response.statusText}`);
    }
    
    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let result = '';
    
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      const chunk = decoder.decode(value);
      result += chunk;
    }
    
    // Parse SSE response
    analysisResult = parseSSE(result);
    renderResults(analysisResult);
    
  } catch (err) {
    showError(err.message);
  } finally {
    analyzeBtn.disabled = false;
    analyzeBtn.textContent = 'Analyze with AI';
  }
});

function parseSSE(text) {
  // Simple SSE parser - split by "data:" prefix
  const lines = text.split('\n');
  let data = '';
  
  for (const line of lines) {
    if (line.startsWith('data:')) {
      data += line.slice(5).trim() + '\n';
    }
  }
  
  // Try to parse as markdown or JSON
  try {
    return { markdown: data, raw: data };
  } catch {
    return { markdown: data, raw: data };
  }
}

function renderResults(result) {
  resultsDiv.style.display = 'block';
  
  // Simple markdown-like rendering
  const html = renderMarkdown(result.markdown || result.raw || 'No results');
  resultsContent.innerHTML = html;
}

function renderMarkdown(text) {
  // Basic markdown rendering
  let html = text
    // Headers
    .replace(/^### (.+)$/gm, '<h4>$1</h4>')
    .replace(/^## (.+)$/gm, '<h3>$1</h3>')
    .replace(/^# (.+)$/gm, '<h2>$1</h2>')
    // Bold
    .replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
    // Lists
    .replace(/^- (.+)$/gm, '<li>$1</li>')
    .replace(/^(\d+)\. (.+)$/gm, '<li>$2</li>')
    // Code blocks
    .replace(/```[\s\S]*?```/g, '<pre>$&</pre>')
    .replace(/`(.+?)`/g, '<code>$1</code>')
    // Line breaks
    .replace(/\n\n/g, '</p><p>')
    .replace(/\n/g, '<br>');
  
  // Wrap loose <li> elements
  html = html.replace(/(<li>.*?<\/li>)+/g, '<ul>$&</ul>');
  
  return `<p>${html}</p>`;
}

function showError(msg) {
  errorDiv.textContent = msg;
  errorDiv.style.display = 'block';
}

// Tab switching
document.querySelectorAll('.tab').forEach(tab => {
  tab.addEventListener('click', () => {
    document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
    tab.classList.add('active');
    // For now, show all content - could be enhanced to filter by tab
  });
});
