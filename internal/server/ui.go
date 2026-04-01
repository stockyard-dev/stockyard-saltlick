package server

import (
	"net/http"
)

// uiHTML is the self-contained dashboard for Saltlick.
// Served at GET /ui — no build step, no external files.
const uiHTML = `<!DOCTYPE html><html lang="en"><head>
<meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Saltlick — Stockyard</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link href="https://fonts.googleapis.com/css2?family=Libre+Baskerville:ital,wght@0,400;0,700;1,400&family=JetBrains+Mono:wght@400;600&display=swap" rel="stylesheet">
<style>:root{
  --bg:#1a1410;--bg2:#241e18;--bg3:#2e261e;
  --rust:#c45d2c;--rust-light:#e8753a;--rust-dark:#8b3d1a;
  --leather:#a0845c;--leather-light:#c4a87a;
  --cream:#f0e6d3;--cream-dim:#bfb5a3;--cream-muted:#7a7060;
  --gold:#d4a843;--green:#5ba86e;--red:#c0392b;
  --font-serif:'Libre Baskerville',Georgia,serif;
  --font-mono:'JetBrains Mono',monospace;
}
*{margin:0;padding:0;box-sizing:border-box}
body{background:var(--bg);color:var(--cream);font-family:var(--font-serif);min-height:100vh;overflow-x:hidden}
a{color:var(--rust-light);text-decoration:none}a:hover{color:var(--gold)}
.hdr{background:var(--bg2);border-bottom:2px solid var(--rust-dark);padding:.9rem 1.8rem;display:flex;align-items:center;justify-content:space-between;gap:1rem}
.hdr-left{display:flex;align-items:center;gap:1rem}
.hdr-brand{font-family:var(--font-mono);font-size:.75rem;color:var(--leather);letter-spacing:3px;text-transform:uppercase}
.hdr-title{font-family:var(--font-mono);font-size:1.1rem;color:var(--cream);letter-spacing:1px}
.badge{font-family:var(--font-mono);font-size:.6rem;padding:.2rem .6rem;letter-spacing:1px;text-transform:uppercase;border:1px solid}
.badge-free{color:var(--green);border-color:var(--green)}
.badge-pro{color:var(--gold);border-color:var(--gold)}
.main{max-width:1000px;margin:0 auto;padding:2rem 1.5rem}
.cards{display:grid;grid-template-columns:repeat(auto-fit,minmax(160px,1fr));gap:1rem;margin-bottom:2rem}
.card{background:var(--bg2);border:1px solid var(--bg3);padding:1.2rem 1.5rem}
.card-val{font-family:var(--font-mono);font-size:1.8rem;font-weight:700;color:var(--cream);display:block}
.card-lbl{font-family:var(--font-mono);font-size:.62rem;letter-spacing:2px;text-transform:uppercase;color:var(--leather);margin-top:.3rem}
.section{margin-bottom:2.5rem}
.section-title{font-family:var(--font-mono);font-size:.68rem;letter-spacing:3px;text-transform:uppercase;color:var(--rust-light);margin-bottom:.8rem;padding-bottom:.5rem;border-bottom:1px solid var(--bg3)}
table{width:100%;border-collapse:collapse;font-family:var(--font-mono);font-size:.78rem}
th{background:var(--bg3);padding:.5rem .8rem;text-align:left;color:var(--leather-light);font-weight:400;letter-spacing:1px;font-size:.65rem;text-transform:uppercase}
td{padding:.5rem .8rem;border-bottom:1px solid var(--bg3);color:var(--cream-dim);vertical-align:top;word-break:break-all}
tr:hover td{background:var(--bg2)}
.empty{color:var(--cream-muted);text-align:center;padding:2rem;font-style:italic}
.btn{font-family:var(--font-mono);font-size:.75rem;padding:.4rem 1rem;border:1px solid var(--leather);background:transparent;color:var(--cream);cursor:pointer;transition:all .2s}
.btn:hover{border-color:var(--rust-light);color:var(--rust-light)}
.btn-rust{border-color:var(--rust);color:var(--rust-light)}.btn-rust:hover{background:var(--rust);color:var(--cream)}
.btn-sm{font-size:.65rem;padding:.25rem .6rem}
.pill{display:inline-block;font-family:var(--font-mono);font-size:.6rem;padding:.1rem .4rem;border-radius:2px;text-transform:uppercase}
.pill-on{background:#1a3a2a;color:var(--green)}.pill-off{background:#2a1a1a;color:var(--red)}
.mono{font-family:var(--font-mono);font-size:.78rem}
.lbl{font-family:var(--font-mono);font-size:.62rem;letter-spacing:1px;text-transform:uppercase;color:var(--leather)}
.upgrade{background:var(--bg2);border:1px solid var(--rust-dark);border-left:3px solid var(--rust);padding:.8rem 1.2rem;font-size:.82rem;color:var(--cream-dim);margin-bottom:1.5rem}
.upgrade a{color:var(--rust-light)}
input,select{font-family:var(--font-mono);font-size:.78rem;background:var(--bg3);border:1px solid var(--bg3);color:var(--cream);padding:.4rem .7rem;outline:none}
input:focus,select:focus{border-color:var(--leather)}
.row{display:flex;gap:.8rem;align-items:flex-end;flex-wrap:wrap;margin-bottom:1rem}
.field{display:flex;flex-direction:column;gap:.3rem}
.toggle{position:relative;width:40px;height:22px;display:inline-block;cursor:pointer}
.toggle input{opacity:0;width:0;height:0}
.toggle .slider{position:absolute;inset:0;background:var(--bg3);border:1px solid var(--cream-muted);transition:.3s}
.toggle .slider:before{content:"";position:absolute;height:16px;width:16px;left:2px;bottom:2px;background:var(--cream-dim);transition:.3s}
.toggle input:checked+.slider{background:var(--green);border-color:var(--green)}
.toggle input:checked+.slider:before{transform:translateX(18px);background:var(--cream)}
.tabs{display:flex;gap:0;margin-bottom:1.5rem;border-bottom:1px solid var(--bg3)}
.tab{font-family:var(--font-mono);font-size:.72rem;padding:.6rem 1.2rem;color:var(--cream-muted);cursor:pointer;border-bottom:2px solid transparent;letter-spacing:1px;text-transform:uppercase}
.tab:hover{color:var(--cream-dim)}
.tab.active{color:var(--rust-light);border-bottom-color:var(--rust-light)}
.tab-content{display:none}.tab-content.active{display:block}
pre{background:var(--bg3);padding:.8rem 1rem;font-family:var(--font-mono);font-size:.72rem;color:var(--cream-dim);overflow-x:auto;max-width:100%}
</style></head><body>
<div class="hdr">
  <div class="hdr-left">
    <svg viewBox="0 0 64 64" width="22" height="22" fill="none"><rect x="8" y="8" width="8" height="48" rx="2.5" fill="#e8753a"/><rect x="28" y="8" width="8" height="48" rx="2.5" fill="#e8753a"/><rect x="48" y="8" width="8" height="48" rx="2.5" fill="#e8753a"/><rect x="8" y="27" width="48" height="7" rx="2.5" fill="#c4a87a"/></svg>
    <span class="hdr-brand">Stockyard</span>
    <span class="hdr-title">Saltlick</span>
  </div>
  <div style="display:flex;gap:.8rem;align-items:center">
    <span id="tier-badge" class="badge badge-free">Free</span>
    <a href="/api/status" class="lbl" style="color:var(--leather)">API</a>
    <a href="https://stockyard.dev/saltlick/" class="lbl" style="color:var(--leather)">Docs</a>
  </div>
</div>
<div class="main">

<div class="cards" id="stat-cards">
  <div class="card"><span class="card-val" id="s-flags">—</span><span class="card-lbl">Flags</span></div>
  <div class="card"><span class="card-val" id="s-enabled">—</span><span class="card-lbl">Enabled</span></div>
  <div class="card"><span class="card-val" id="s-evals">—</span><span class="card-lbl">Evaluations</span></div>
</div>

<div class="tabs">
  <div class="tab active" onclick="switchTab('flags')">Flags</div>
  <div class="tab" onclick="switchTab('create')">Create</div>
  <div class="tab" onclick="switchTab('test')">Test</div>
  <div class="tab" onclick="switchTab('usage')">Usage</div>
</div>

<!-- Flags Tab -->
<div id="tab-flags" class="tab-content active">
  <div class="section">
    <div class="section-title">Feature Flags</div>
    <table><thead><tr>
      <th>Name</th><th>Status</th><th>Rollout</th><th>Environment</th><th>Updated</th><th></th>
    </tr></thead><tbody id="flags-body"></tbody></table>
  </div>
</div>

<!-- Create Tab -->
<div id="tab-create" class="tab-content">
  <div class="section">
    <div class="section-title">Create Flag</div>
    <div class="row">
      <div class="field"><span class="lbl">Name</span><input id="c-name" placeholder="new-checkout" style="width:200px"></div>
      <div class="field"><span class="lbl">Description</span><input id="c-desc" placeholder="New checkout flow" style="width:260px"></div>
      <div class="field"><span class="lbl">Enabled</span>
        <label class="toggle"><input type="checkbox" id="c-enabled"><span class="slider"></span></label>
      </div>
      <button class="btn btn-rust" onclick="createFlag()">Create</button>
    </div>
    <div id="c-result" style="margin-top:.5rem"></div>
  </div>
</div>

<!-- Test Tab -->
<div id="tab-test" class="tab-content">
  <div class="section">
    <div class="section-title">Evaluate Flag</div>
    <div class="row">
      <div class="field"><span class="lbl">Flag name</span><input id="t-name" placeholder="new-checkout" style="width:180px"></div>
      <div class="field"><span class="lbl">User ID (optional)</span><input id="t-user" placeholder="user_123" style="width:150px"></div>
      <button class="btn btn-rust" onclick="testEval()">Evaluate</button>
    </div>
    <pre id="t-result" style="margin-top:.8rem;display:none"></pre>
  </div>
</div>

<!-- Usage Tab -->
<div id="tab-usage" class="tab-content">
  <div class="section">
    <div class="section-title">Quick Start</div>
    <pre id="usage-block">
# Create a flag
curl -X POST http://localhost:8800/api/flags \
  -H "Content-Type: application/json" \
  -d '{"name":"new-checkout","description":"New checkout flow","enabled":true}'

# Evaluate a flag
curl http://localhost:8800/api/eval/new-checkout?user_id=user_123

# Batch evaluate
curl -X POST http://localhost:8800/api/eval/batch \
  -H "Content-Type: application/json" \
  -d '{"flags":["new-checkout","dark-mode"],"user_id":"user_123"}'

# Update flag (enable percentage rollout)
curl -X PUT http://localhost:8800/api/flags/new-checkout \
  -H "Content-Type: application/json" \
  -d '{"rollout_percent":25}'

# Get flag stats
curl http://localhost:8800/api/flags/new-checkout/stats
    </pre>
  </div>
</div>

</div>
<script>
let flags=[];

function switchTab(name){
  document.querySelectorAll('.tab').forEach((t,i)=>t.classList.toggle('active',t.textContent.toLowerCase().replace(/\s/g,'')===name));
  document.querySelectorAll('.tab-content').forEach(t=>t.classList.toggle('active',t.id==='tab-'+name));
}

async function refresh(){
  try{
    const sr=await fetch('/api/status');const st=await sr.json();
    document.getElementById('s-flags').textContent=st.flags||0;
    document.getElementById('s-enabled').textContent=st.enabled_flags||0;
    document.getElementById('s-evals').textContent=fmt(st.evaluations||0);
  }catch(e){}
  try{
    const fr=await fetch('/api/flags');const fd=await fr.json();
    flags=fd.flags||[];
    const tb=document.getElementById('flags-body');
    if(!flags.length){tb.innerHTML='<tr><td colspan="6" class="empty">No flags yet. Create one to get started.</td></tr>';return;}
    tb.innerHTML=flags.map(f=>{
      const on=f.enabled;
      return '<tr>'+
        '<td style="color:var(--cream);font-weight:600">'+esc(f.name)+'<br><span style="font-size:.6rem;color:var(--cream-muted)">'+esc(f.description||'')+'</span></td>'+
        '<td><span class="pill '+(on?'pill-on':'pill-off')+'">'+(on?'ON':'OFF')+'</span></td>'+
        '<td>'+(f.rollout_percent||0)+'%</td>'+
        '<td>'+esc(f.environment||'production')+'</td>'+
        '<td style="font-size:.68rem;color:var(--cream-muted)">'+timeAgo(f.updated_at)+'</td>'+
        '<td><button class="btn btn-sm" onclick="toggleFlag(\''+esc(f.name)+'\','+(!on)+')">'+(!on?'Enable':'Disable')+'</button> '+
        '<button class="btn btn-sm" onclick="deleteFlag(\''+esc(f.name)+'\')">Delete</button></td>'+
        '</tr>';
    }).join('');
  }catch(e){}
}

async function createFlag(){
  const name=document.getElementById('c-name').value.trim();
  const desc=document.getElementById('c-desc').value.trim();
  const enabled=document.getElementById('c-enabled').checked;
  if(!name){document.getElementById('c-result').innerHTML='<span style="color:var(--red)">Name is required</span>';return;}
  try{
    const r=await fetch('/api/flags',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({name,description:desc,enabled})});
    const d=await r.json();
    if(r.ok){
      document.getElementById('c-result').innerHTML='<span style="color:var(--green)">Flag "'+esc(name)+'" created</span>';
      document.getElementById('c-name').value='';document.getElementById('c-desc').value='';
      refresh();
    }else{document.getElementById('c-result').innerHTML='<span style="color:var(--red)">'+esc(d.error)+'</span>';}
  }catch(e){document.getElementById('c-result').innerHTML='<span style="color:var(--red)">Error: '+e.message+'</span>';}
}

async function toggleFlag(name,enabled){
  await fetch('/api/flags/'+name,{method:'PUT',headers:{'Content-Type':'application/json'},body:JSON.stringify({enabled})});
  refresh();
}

async function deleteFlag(name){
  if(!confirm('Delete flag "'+name+'"?'))return;
  await fetch('/api/flags/'+name,{method:'DELETE'});
  refresh();
}

async function testEval(){
  const name=document.getElementById('t-name').value.trim();
  const user=document.getElementById('t-user').value.trim();
  if(!name){return;}
  const url='/api/eval/'+encodeURIComponent(name)+(user?'?user_id='+encodeURIComponent(user):'');
  try{
    const r=await fetch(url);const d=await r.json();
    const el=document.getElementById('t-result');
    el.style.display='block';
    el.textContent=JSON.stringify(d,null,2);
  }catch(e){}
}

function fmt(n){if(n>=1e6)return(n/1e6).toFixed(1)+'M';if(n>=1e3)return(n/1e3).toFixed(1)+'K';return n;}
function esc(s){const d=document.createElement('div');d.textContent=s;return d.innerHTML;}
function timeAgo(s){if(!s)return'—';const d=new Date(s);const diff=Date.now()-d.getTime();if(diff<60000)return'just now';if(diff<3600000)return Math.floor(diff/60000)+'m ago';if(diff<86400000)return Math.floor(diff/3600000)+'h ago';return Math.floor(diff/86400000)+'d ago';}

refresh();
setInterval(refresh,8000);
</script></body></html>`

func (s *Server) handleUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(uiHTML))
}
