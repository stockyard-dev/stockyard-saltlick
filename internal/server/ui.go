package server

import "net/http"

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(dashboardHTML))
}

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1.0">
<title>Salt Lick</title>
<style>
:root{--bg:#1a1410;--bg2:#241e18;--bg3:#2e261e;--rust:#e8753a;--leather:#a0845c;--cream:#f0e6d3;--cd:#bfb5a3;--cm:#7a7060;--gold:#d4a843;--green:#4a9e5c;--red:#c94444;--mono:'JetBrains Mono',monospace;--serif:'Libre Baskerville',serif}
*{margin:0;padding:0;box-sizing:border-box}body{background:var(--bg);color:var(--cream);font-family:var(--serif);line-height:1.6}
.header{padding:1rem 1.5rem;border-bottom:1px solid var(--bg3);display:flex;justify-content:space-between;align-items:center}
.header h1{font-family:var(--mono);font-size:.9rem;letter-spacing:2px}
.stats{font-family:var(--mono);font-size:.7rem;color:var(--cm)}
.tabs{display:flex;border-bottom:1px solid var(--bg3);padding:0 1.5rem;font-family:var(--mono);font-size:.75rem}
.tab{padding:.7rem 1.2rem;cursor:pointer;color:var(--cm);border-bottom:2px solid transparent}.tab:hover{color:var(--cream)}.tab.active{color:var(--rust);border-color:var(--rust)}
.content{padding:1.5rem;max-width:900px;margin:0 auto}
.flag-card{background:var(--bg2);border:1px solid var(--bg3);margin-bottom:.6rem;padding:1rem 1.2rem}
.flag-top{display:flex;justify-content:space-between;align-items:flex-start}
.flag-info{flex:1}
.flag-key{font-family:var(--mono);font-size:.82rem;color:var(--cream)}
.flag-name{font-size:.78rem;color:var(--cd);margin-top:.1rem}
.flag-desc{font-size:.75rem;color:var(--cm);margin-top:.2rem}
.flag-meta{display:flex;gap:.8rem;margin-top:.4rem;font-family:var(--mono);font-size:.6rem;color:var(--cm)}
.flag-controls{display:flex;flex-direction:column;align-items:flex-end;gap:.5rem;flex-shrink:0;margin-left:1rem}
.toggle{position:relative;width:42px;height:22px;cursor:pointer}
.toggle input{opacity:0;width:0;height:0}
.toggle .slider{position:absolute;inset:0;background:var(--bg3);border-radius:11px;transition:.2s}
.toggle .slider:before{content:'';position:absolute;width:16px;height:16px;left:3px;bottom:3px;background:var(--cm);border-radius:50%;transition:.2s}
.toggle input:checked+.slider{background:var(--green)}
.toggle input:checked+.slider:before{transform:translateX(20px);background:var(--cream)}
.rollout-bar{display:flex;align-items:center;gap:.5rem;font-family:var(--mono);font-size:.65rem;color:var(--cm)}
.rollout-bar input[type=range]{width:100px;accent-color:var(--rust);height:4px}
.rollout-pct{min-width:28px;text-align:right}
.env-badge{font-family:var(--mono);font-size:.55rem;padding:.1rem .4rem;border:1px solid var(--bg3);color:var(--cm)}
.tag{font-family:var(--mono);font-size:.55rem;padding:.1rem .3rem;background:var(--bg3);color:var(--cm);margin-right:.2rem}
.btn{font-family:var(--mono);font-size:.65rem;padding:.3rem .7rem;cursor:pointer;border:1px solid var(--bg3);background:var(--bg);color:var(--cd)}.btn:hover{border-color:var(--leather);color:var(--cream)}
.btn-primary{background:var(--rust);border-color:var(--rust);color:var(--bg)}.btn-primary:hover{opacity:.85}
.btn-sm{font-size:.6rem;padding:.2rem .5rem}
.modal-bg{display:none;position:fixed;inset:0;background:rgba(0,0,0,.6);z-index:100;align-items:center;justify-content:center}
.modal-bg.open{display:flex}
.modal{background:var(--bg2);border:1px solid var(--bg3);padding:1.5rem;width:420px;max-width:90vw}
.modal h2{font-family:var(--mono);font-size:.8rem;margin-bottom:1rem;color:var(--rust)}
.form-row{margin-bottom:.7rem}
.form-row label{display:block;font-family:var(--mono);font-size:.6rem;color:var(--cm);text-transform:uppercase;letter-spacing:1px;margin-bottom:.2rem}
.form-row input,.form-row select,.form-row textarea{width:100%;padding:.45rem .6rem;background:var(--bg);border:1px solid var(--bg3);color:var(--cream);font-family:var(--mono);font-size:.78rem}
.actions{display:flex;gap:.5rem;justify-content:flex-end;margin-top:1rem}
.log-entry{display:flex;gap:.8rem;padding:.4rem 0;border-bottom:1px solid var(--bg3);font-family:var(--mono);font-size:.7rem}
.log-time{color:var(--cm);font-size:.6rem;flex-shrink:0;width:100px}
.log-action{color:var(--cd)}
.log-action .created{color:var(--green)}.log-action .enabled{color:var(--green)}.log-action .disabled{color:var(--red)}.log-action .deleted{color:var(--red)}.log-action .rollout_changed{color:var(--gold)}
.empty{text-align:center;padding:3rem;color:var(--cm);font-style:italic}
</style>
</head>
<body>
<div class="header">
  <h1>SALT LICK</h1>
  <div class="stats" id="statsBar"></div>
</div>
<div class="tabs">
  <div class="tab active" onclick="showTab('flags')">Flags</div>
  <div class="tab" onclick="showTab('log')">Audit Log</div>
</div>
<div class="content" id="main"></div>
<div class="modal-bg" id="modalBg" onclick="if(event.target===this)closeModal()"><div class="modal" id="modal"></div></div>

<script>
const API='/api';
let flags=[],log=[],tab='flags';

async function load(){
  const[f,st]=await Promise.all([
    fetch(API+'/flags').then(r=>r.json()),
    fetch(API+'/stats').then(r=>r.json()),
  ]);
  flags=f.flags||[];
  document.getElementById('statsBar').innerHTML=st.enabled+' enabled / '+st.disabled+' disabled / '+st.total+' total';
  render();
}

function showTab(t){
  tab=t;
  document.querySelectorAll('.tab').forEach((el,i)=>el.classList.toggle('active',['flags','log'][i]===t));
  if(t==='log')loadLog();
  else render();
}

async function loadLog(){
  const r=await fetch(API+'/log').then(r=>r.json());
  log=r.log||[];render();
}

function render(){
  const m=document.getElementById('main');
  if(tab==='log'){renderLog(m);return;}
  let h='<div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:1rem"><h2 style="font-family:var(--mono);font-size:.75rem;color:var(--leather)">FEATURE FLAGS</h2><button class="btn btn-primary" onclick="openForm()">+ New Flag</button></div>';
  if(!flags.length){h+='<div class="empty">No flags yet. Create your first feature flag.</div>';}
  else{flags.forEach(f=>{
    const tags=f.tags?f.tags.split(',').map(t=>'<span class="tag">'+esc(t.trim())+'</span>').join(''):'';
    h+='<div class="flag-card"><div class="flag-top"><div class="flag-info"><div class="flag-key">'+esc(f.key)+'</div>';
    if(f.name)h+='<div class="flag-name">'+esc(f.name)+'</div>';
    if(f.description)h+='<div class="flag-desc">'+esc(f.description)+'</div>';
    h+='<div class="flag-meta">';
    h+='<span class="env-badge">'+f.environment+'</span>';
    if(tags)h+=tags;
    h+='<span>updated '+fmtTime(f.updated_at)+'</span>';
    h+='</div></div>';
    h+='<div class="flag-controls"><label class="toggle"><input type="checkbox" '+(f.enabled?'checked':'')+' onchange="toggleFlag(\''+f.id+'\',this.checked)"><span class="slider"></span></label>';
    h+='<div class="rollout-bar"><input type="range" min="0" max="100" value="'+f.rollout+'" onchange="setRollout(\''+f.id+'\',parseInt(this.value));this.nextElementSibling.textContent=this.value+\'%\'"><span class="rollout-pct">'+f.rollout+'%</span></div>';
    h+='<button class="btn btn-sm" onclick="delFlag(\''+f.id+'\')" style="color:var(--red);font-size:.55rem">Delete</button>';
    h+='</div></div></div>';
  });}
  m.innerHTML=h;
}

function renderLog(m){
  let h='<h2 style="font-family:var(--mono);font-size:.75rem;color:var(--leather);margin-bottom:1rem">AUDIT LOG</h2>';
  if(!log.length){h+='<div class="empty">No flag changes recorded yet.</div>';}
  else{log.forEach(l=>{
    h+='<div class="log-entry"><span class="log-time">'+fmtTime(l.created_at)+'</span><span class="log-action"><span class="'+l.action+'">'+l.action+'</span> <strong>'+esc(l.flag_key)+'</strong>';
    if(l.detail)h+=' — '+esc(l.detail);
    h+='</span></div>';
  });}
  m.innerHTML=h;
}

async function toggleFlag(id,enabled){
  await fetch(API+'/flags/'+id+'/toggle',{method:'PATCH',headers:{'Content-Type':'application/json'},body:JSON.stringify({enabled})});
  load();
}

async function setRollout(id,pct){
  await fetch(API+'/flags/'+id+'/rollout',{method:'PATCH',headers:{'Content-Type':'application/json'},body:JSON.stringify({rollout:pct})});
}

async function delFlag(id){if(confirm('Delete this flag?')){await fetch(API+'/flags/'+id,{method:'DELETE'});load();}}

function openForm(){
  document.getElementById('modal').innerHTML='<h2>New Feature Flag</h2><div class="form-row"><label>Key (unique identifier)</label><input id="f-key" placeholder="e.g. new-dashboard"></div><div class="form-row"><label>Name</label><input id="f-name" placeholder="e.g. New Dashboard UI"></div><div class="form-row"><label>Description</label><input id="f-desc" placeholder="What this flag controls"></div><div class="form-row"><label>Environment</label><select id="f-env"><option value="all">All</option><option value="development">Development</option><option value="staging">Staging</option><option value="production">Production</option></select></div><div class="form-row"><label>Rollout %</label><input id="f-rollout" type="number" min="0" max="100" value="100"></div><div class="form-row"><label>Tags (comma separated)</label><input id="f-tags" placeholder="e.g. frontend, experiment"></div><div class="form-row"><label><input type="checkbox" id="f-enabled" checked> Enabled on creation</label></div><div class="actions"><button class="btn" onclick="closeModal()">Cancel</button><button class="btn btn-primary" onclick="submitFlag()">Create</button></div>';
  document.getElementById('modalBg').classList.add('open');
}

async function submitFlag(){
  await fetch(API+'/flags',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({key:document.getElementById('f-key').value,name:document.getElementById('f-name').value,description:document.getElementById('f-desc').value,environment:document.getElementById('f-env').value,rollout:parseInt(document.getElementById('f-rollout').value)||100,tags:document.getElementById('f-tags').value,enabled:document.getElementById('f-enabled').checked})});
  closeModal();load();
}

function closeModal(){document.getElementById('modalBg').classList.remove('open');}
function esc(s){if(!s)return'';const d=document.createElement('div');d.textContent=s;return d.innerHTML;}
function fmtTime(t){if(!t)return'';const d=new Date(t);return d.toLocaleDateString()+' '+d.toLocaleTimeString([],{hour:'2-digit',minute:'2-digit'});}

load();
</script>
</body>
</html>`
