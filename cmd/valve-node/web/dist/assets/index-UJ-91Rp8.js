var oe=Object.defineProperty;var ie=(t,a,n)=>a in t?oe(t,a,{enumerable:!0,configurable:!0,writable:!0,value:n}):t[a]=n;var V=(t,a,n)=>ie(t,typeof a!="symbol"?a+"":a,n);(function(){const a=document.createElement("link").relList;if(a&&a.supports&&a.supports("modulepreload"))return;for(const c of document.querySelectorAll('link[rel="modulepreload"]'))e(c);new MutationObserver(c=>{for(const l of c)if(l.type==="childList")for(const b of l.addedNodes)b.tagName==="LINK"&&b.rel==="modulepreload"&&e(b)}).observe(document,{childList:!0,subtree:!0});function n(c){const l={};return c.integrity&&(l.integrity=c.integrity),c.referrerPolicy&&(l.referrerPolicy=c.referrerPolicy),c.crossOrigin==="use-credentials"?l.credentials="include":c.crossOrigin==="anonymous"?l.credentials="omit":l.credentials="same-origin",l}function e(c){if(c.ep)return;c.ep=!0;const l=n(c);fetch(c.href,l)}})();function U(){return R("/api/catalog")}function O(){return R("/api/targets")}function Y(t){return R("/api/targets",{method:"POST",headers:j,body:JSON.stringify(t)})}function ce(t){return R(`/api/targets/${encodeURIComponent(t)}`,{method:"DELETE"})}function le(t,a){return R(`/api/targets/${encodeURIComponent(t)}/setup`,{method:"POST",headers:j,body:JSON.stringify(a)})}function de(t,a){const n=new EventSource(`/api/targets/${encodeURIComponent(t)}/setup/stream`);return n.onmessage=e=>{try{a(JSON.parse(e.data))}catch{}},()=>n.close()}function ue(t,a){const n=new EventSource(`/api/targets/${encodeURIComponent(t)}/monitor/stream`);return n.onmessage=e=>{try{a(JSON.parse(e.data))}catch{}},()=>n.close()}function pe(t,a=200){return R(`/api/targets/${encodeURIComponent(t)}/logs?n=${a}`)}function he(t,a){const n=new EventSource(`/api/targets/${encodeURIComponent(t)}/logs/stream`);return n.onmessage=e=>{try{a(JSON.parse(e.data))}catch{}},()=>n.close()}function Q(t,a){const n=a===void 0?{}:{lines:a};return R(`/api/targets/${encodeURIComponent(t)}/explain`,{method:"POST",headers:j,body:JSON.stringify(n)})}function fe(){return R("/api/settings")}function me(t){return R("/api/settings",{method:"PUT",headers:j,body:JSON.stringify(t)})}class z extends Error{constructor(n,e){super(e);V(this,"status");this.name="ApiError",this.status=n}}const j={"Content-Type":"application/json"};async function R(t,a){const n=await fetch(t,a);if(!n.ok){let c=n.statusText||`HTTP ${n.status}`;try{const l=await n.json();l&&typeof l.error=="string"&&l.error&&(c=l.error)}catch{}throw new z(n.status,c)}if(n.status===204)return;const e=await n.text();return e?JSON.parse(e):void 0}const ve="https://learn.valve.city/rpc";function o(t){return t.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;").replace(/'/g,"&#39;")}function C(t,a){const n=t&&a?` <span class="footer-sep">·</span> <a href="${o(a)}" target="_blank" rel="noopener noreferrer">${o(t)}</a>`:"";return`
    <footer class="footer">
      <a href="${o(ve)}" target="_blank" rel="noopener noreferrer">Learn how this works → learn.valve.city/rpc</a>${n}
    </footer>
  `}function ge(t){t.innerHTML=`
    <div class="shell">
      <header class="topbar">
        <a class="brand" href="#/targets">valve-node</a>
        <nav class="nav">
          <a href="#/targets" data-nav="targets">Targets</a>
          <a href="#/settings" data-nav="settings">Settings</a>
        </nav>
      </header>
      <main id="content" class="content"></main>
    </div>
  `;const a=t.querySelector("#content"),n=Array.from(t.querySelectorAll("[data-nav]"));return{contentEl:a,setActiveNav:c=>{for(const l of n)l.classList.toggle("active",l.dataset.nav===c)}}}function H(t){return Number.isFinite(t)?t.toLocaleString("en-US"):"—"}function be(t){return Number.isFinite(t)?`${t.toFixed(1)}%`:"—"}function ye(t){if(!Number.isFinite(t)||t<0)return"—";if(t<60)return`~${Math.round(t)}s`;const a=Math.round(t/60),n=Math.floor(a/60),e=a%60;if(n===0)return`~${e}m`;if(n<48)return`~${n}h ${e}m`;const c=Math.floor(n/24),l=n%24;return`~${c}d ${l}h`}function x(t,a){return`<span class="badge badge-${a}">${o(t)}</span>`}function B(t,a){t.addEventListener("click",n=>{const e=n.target.closest("[data-action]");if(!e||!t.contains(e))return;const c=e.dataset.action;c&&a(c,e,n)})}const $e=85;function we(t,a){let n=!1,e=null,c=null,l=null;t.innerHTML=`<h1>Dashboard: ${o(a)}</h1><div id="dash-body"><p class="muted">Loading…</p></div><div id="dash-footer">${C()}</div>`;const b=t.querySelector("#dash-body"),w=t.querySelector("#dash-footer");L();async function L(){let r,s;try{const[$,A]=await Promise.all([O(),U()]);r=$.find(u=>u.id===a),s=A}catch($){if(n)return;b.innerHTML=`<p class="error">Failed to load target: ${o(String($))}</p>`;return}if(n)return;if(!r){b.innerHTML=`<p class="error">Target "${o(a)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!r.wire){b.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(a)}">Run the setup wizard →</a></p>`;return}const f=s==null?void 0:s.networks.find($=>$.ChainID===r.wire.ChainID);f&&(w.innerHTML=C(f.Name,f.LearnURL)),b.innerHTML='<p class="muted">Connecting…</p>',e=ue(a,$=>{n||(k($),T($),c=$)})}function k(r){if(!c)return;const s=(new Date(r.at).getTime()-new Date(c.at).getTime())/1e3,f=r.execHead-c.execHead;if(s>0&&f>=0){const $=f/s;l=l===null?$:l*.7+$*.3}}function T(r){b.innerHTML=`
      <div class="card-grid">
        ${d(r)}
        ${m(r)}
        ${y(r)}
        ${S(r)}
        ${i(r)}
      </div>
      <p class="muted small">Last updated ${o(new Date(r.at).toLocaleTimeString())}</p>
    `}function d(r){const s=r.refHead>0,f=s?r.refHead-r.execHead:null,$=f!==null&&f>0&&l&&l>0?ye(f/l):f!==null&&f<=0?"caught up":"—";return`
      <div class="card">
        <h3>Execution sync</h3>
        <p>${r.execSyncing?x("syncing","warn"):x("synced","ok")}</p>
        <dl class="stat-list">
          <div><dt>Local head</dt><dd>${H(r.execHead)}</dd></div>
          <div><dt>Reference head</dt><dd>${s?H(r.refHead):"unavailable"}</dd></div>
          <div><dt>Lag</dt><dd>${f!==null?H(Math.max(f,0))+" blocks":"—"}</dd></div>
          <div><dt>ETA</dt><dd>${$}</dd></div>
        </dl>
      </div>
    `}function m(r){return`
      <div class="card">
        <h3>Beacon sync</h3>
        <p>${r.beaconDistance===0?x("synced","ok"):x("syncing","warn")}</p>
        <dl class="stat-list">
          <div><dt>Slot</dt><dd>${H(r.beaconSlot)}</dd></div>
          <div><dt>Distance</dt><dd>${H(r.beaconDistance)}</dd></div>
        </dl>
      </div>
    `}function y(r){return`
      <div class="card">
        <h3>Peers</h3>
        <dl class="stat-list">
          <div><dt>Execution</dt><dd>${H(r.execPeers)}</dd></div>
          <div><dt>Beacon</dt><dd>${H(r.beaconPeers)}</dd></div>
        </dl>
      </div>
    `}function S(r){const s=r.diskUsedPct>=$e;return`
      <div class="card ${s?"card-warn":""}">
        <h3>Disk</h3>
        <div class="meter"><div class="meter-fill ${s?"meter-warn":""}" style="width:${Math.min(r.diskUsedPct,100)}%"></div></div>
        <p>${be(r.diskUsedPct)} used</p>
      </div>
    `}function i(r){return`
      <div class="card">
        <h3>Services</h3>
        <p>Execution ${r.execActive?x("active","ok"):x("down","bad")}</p>
        <p>Beacon ${r.beaconActive?x("active","ok"):x("down","bad")}</p>
        <p><a href="#/logs/${encodeURIComponent(a)}">View logs →</a></p>
      </div>
    `}return()=>{n=!0,e==null||e()}}const X=500,Z="valve-node.explain-consent";function Se(t,a){let n=!1,e=null;const c=[];t.innerHTML=`
    <h1>Logs: ${o(a)}</h1>
    <div id="logs-body"><p class="muted">Loading…</p></div>
    <div id="logs-footer">${C()}</div>
  `;const l=t.querySelector("#logs-body"),b=t.querySelector("#logs-footer");B(t,i=>{i==="explain"&&T()}),w();async function w(){let i,r;try{const[f,$]=await Promise.all([O(),U()]);i=f.find(A=>A.id===a),r=$}catch(f){if(n)return;l.innerHTML=`<p class="error">Failed to load target: ${o(String(f))}</p>`;return}if(n)return;if(!i){l.innerHTML=`<p class="error">Target "${o(a)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!i.wire){l.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(a)}">Run the setup wizard →</a></p>`;return}const s=r==null?void 0:r.networks.find(f=>f.ChainID===i.wire.ChainID);s&&(b.innerHTML=C(s.Name,s.LearnURL));try{const f=await pe(a,200);if(n)return;c.push(...f)}catch(f){if(n)return;l.innerHTML=`<p class="error">Failed to load logs: ${o(String(f))}</p>`;return}L(),e=he(a,f=>{n||(c.push(f),c.length>X&&c.splice(0,c.length-X),L())})}function L(){const i=c.filter(s=>s.severity==="error"||s.severity==="critical");l.innerHTML=`
      <div class="logs-layout">
        <section class="logs-tail">
          <div class="logs-tail-head">
            <h2>Live tail</h2>
            <button class="btn" data-action="explain">Explain with AI</button>
          </div>
          <div class="log-lines">${c.map(k).join("")}</div>
        </section>
        <section class="logs-errors">
          <h2>Error feed ${x(String(i.length),i.length?"bad":"neutral")}</h2>
          <div class="log-lines">${i.length?i.slice().reverse().map(k).join(""):'<p class="muted">No errors seen yet.</p>'}</div>
        </section>
      </div>
    `;const r=l.querySelector(".log-lines");r&&(r.scrollTop=r.scrollHeight)}function k(i){const r=i.severity||"info",s=i.learnUrl?` <a href="${o(i.learnUrl)}" target="_blank" rel="noopener noreferrer">learn →</a>`:"";return`
      <div class="log-line log-${o(r)}">
        <span class="log-time">${o(new Date(i.at).toLocaleTimeString())}</span>
        <span class="log-unit">${o(i.unit)}</span>
        <span class="log-sev">${o(r)}</span>
        <span class="log-text">${o(i.line)}</span>
        ${i.explain?`<div class="log-explain">${o(i.explain)}${s}</div>`:""}
      </div>
    `}async function T(){const i=c.filter(s=>s.severity==="error"||s.severity==="critical").map(s=>s.line).slice(-40);if(!(localStorage.getItem(Z)==="1")){d(i);return}await m(i)}function d(i){const r=i.length?`<pre class="explain-excerpt">${i.map(s=>o(s)).join(`
`)}</pre>`:'<p class="muted">No recent error lines are loaded yet — the server will auto-select its own recent error/critical lines instead.</p>';y(`
      <h2>Send logs to your AI provider?</h2>
      <p>
        The excerpt below will be sent to the AI provider configured in
        <a href="#/settings">Settings</a> to generate a plain-English
        explanation. This happens every time you click "Explain with AI";
        this confirmation only shows once per browser.
      </p>
      ${r}
      <div class="modal-actions">
        <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        <button class="btn btn-primary" data-modal-action="proceed">Send to AI provider</button>
      </div>
    `,s=>{s==="proceed"?(localStorage.setItem(Z,"1"),S(),m(i)):S()})}async function m(i){y('<h2>Explain with AI</h2><p class="muted">Asking the AI provider…</p>',()=>{});try{const r=i.length?await Q(a,i):await Q(a);if(n)return;y(`
        <h2>Explanation</h2>
        <div class="explain-text">${o(r.text)}</div>
        <details class="advanced">
          <summary>What was sent</summary>
          <pre class="explain-excerpt">${r.sentExcerpt.map(s=>o(s)).join(`
`)||"(no log lines — general question only)"}</pre>
        </details>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,s=>{s==="close"&&S()})}catch(r){if(n)return;if(r instanceof z&&r.status===409){y(`
          <h2>No AI provider configured</h2>
          <p>Set a provider and key in <a href="#/settings">Settings</a>, then try again.</p>
          <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
        `,s=>{s==="close"&&S()});return}y(`
        <h2>Explain failed</h2>
        <p class="error">${o(r instanceof Error?r.message:String(r))}</p>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,s=>{s==="close"&&S()})}}function y(i,r){S();const s=document.createElement("div");s.className="modal-overlay",s.id="explain-modal",s.innerHTML=`<div class="modal">${i}</div>`,s.addEventListener("click",f=>{const $=f.target.closest("[data-modal-action]");$!=null&&$.dataset.modalAction&&r($.dataset.modalAction),f.target===s&&r("cancel")}),document.body.appendChild(s)}function S(){var i;(i=document.getElementById("explain-modal"))==null||i.remove()}return()=>{n=!0,e==null||e(),S()}}const ke=[{value:"",label:"None"},{value:"gemini",label:"Gemini"},{value:"groq",label:"Groq"},{value:"ollama",label:"Ollama"}];function xe(t){let a=!1,n=!1,e=!1,c=null,l=!1,b=null;t.innerHTML=`<h1>Settings</h1><div id="settings-body"><p class="muted">Loading…</p></div>${C()}`;const w=t.querySelector("#settings-body");B(t,d=>{if(d==="save"&&T(),d==="clear-key"){if(!b)return;n=!0;const m=t.querySelector("#ai-key");m&&(m.value=""),k(b)}}),L();async function L(){try{const d=await fe();if(a)return;b=d,k(d)}catch(d){if(a)return;w.innerHTML=`<p class="error">Failed to load settings: ${o(String(d))}</p>`}}function k(d){var S,i;const m=ke.map(r=>`<option value="${r.value}" ${d.aiProvider===r.value?"selected":""}>${o(r.label)}</option>`).join("");w.innerHTML=`
      <form class="card" id="settings-form" onsubmit="return false">
        <label>
          AI provider
          <select id="ai-provider">${m}</select>
        </label>
        <label>
          API key
          <input id="ai-key" type="password" placeholder="${d.aiKeySet?"•••••••• (leave blank to keep)":"no key set"}" autocomplete="off" />
        </label>
        ${d.aiKeySet?'<button class="btn btn-ghost" type="button" data-action="clear-key">Clear saved key</button>':""}
        <p class="muted small">Keys stay on this machine — they're written to ~/.valve-node/config.json (mode 0600) and only sent to the provider you pick, never anywhere else.</p>
        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            Reference RPC base
            <input id="ref-rpc-base" type="text" value="${o(d.refRpcBase)}" />
          </label>
          <p class="muted small">Used to compute head-lag on the dashboard. Leave the default unless you have your own reference endpoint.</p>
        </details>
        ${c?`<p class="error">${o(c)}</p>`:""}
        ${l?'<p class="ok">Saved.</p>':""}
        <button class="btn btn-primary" type="button" data-action="save" ${e?"disabled":""}>${e?"Saving…":"Save"}</button>
      </form>
    `;const y=t.querySelector("#ai-key");y==null||y.addEventListener("input",()=>{n=!0,l=!1}),(S=t.querySelector("#ai-provider"))==null||S.addEventListener("change",()=>{l=!1}),(i=t.querySelector("#ref-rpc-base"))==null||i.addEventListener("input",()=>{l=!1})}async function T(){const d=t.querySelector("#ai-provider"),m=t.querySelector("#ai-key"),y=t.querySelector("#ref-rpc-base");if(!d||!m||!y||!b)return;const S={aiProvider:d.value,refRpcBase:y.value.trim()};n&&(S.aiKey=m.value),e=!0,c=null,l=!1,k(b);try{const i=await me(S);if(a)return;b=i,n=!1,e=!1,l=!0,k(i)}catch(i){if(a)return;e=!1,c=String(i instanceof Error?i.message:i),k(b)}}return()=>{a=!0}}const Ie="local";function Ee(t){let a=!1;t.innerHTML=`
    <h1>Targets</h1>
    <div id="targets-body"><p class="muted">Loading…</p></div>
    ${C()}
  `;const n=t.querySelector("#targets-body");B(t,(d,m)=>{l(d,m)}),e();async function e(){try{const[d,m]=await Promise.all([O(),U()]);if(a)return;c(d,m)}catch(d){if(a)return;n.innerHTML=`<p class="error">Failed to load targets: ${o(String(d))}</p>`}}function c(d,m){const y=d.find(s=>s.mode==="local"),S=d.filter(s=>s.mode==="ssh"),i=y?ee(y,m):`
        <div class="card">
          <h2>This machine</h2>
          <p class="muted">${Ce()}</p>
          <button class="btn" data-action="add-local">Add this machine as a target</button>
        </div>
      `,r=S.length?S.map(s=>ee(s,m)).join(""):'<p class="muted">No SSH targets yet.</p>';n.innerHTML=`
      <section class="section">${i}</section>
      <section class="section">
        <h2>Servers over SSH</h2>
        <div class="card-grid">${r}</div>
        ${Le()}
      </section>
    `}async function l(d,m){if(d==="add-local"){await b();return}if(d==="delete-target"){const y=m.dataset.id;if(!y||!confirm(`Remove target "${y}"? This does not touch anything already running on it.`))return;await w(y);return}d==="add-ssh"&&await L()}async function b(){T();try{await Y({id:Ie,mode:"local"}),await e()}catch(d){k(d)}}async function w(d){try{await ce(d),await e()}catch(m){k(m)}}async function L(){const d=t.querySelector("#ssh-host"),m=t.querySelector("#ssh-user"),y=t.querySelector("#ssh-key"),S=t.querySelector("#ssh-port"),i=t.querySelector("#ssh-id");if(!d||!m||!y||!S||!i)return;const r=d.value.trim(),s=m.value.trim(),f=y.value.trim(),$=S.value.trim(),A=i.value.trim();if(T(),!r||!s||!f){k(new Error("host, user, and key path are required"));return}const u=A||Re(r),h={Host:r,User:s,KeyPath:f};if($){const v=Number.parseInt($,10);if(!Number.isFinite(v)||v<=0){k(new Error("port must be a positive number"));return}h.Port=v}const p=t.querySelector("#ssh-submit");p&&(p.disabled=!0,p.textContent="Connecting…");try{await Y({id:u,mode:"ssh",ssh:h}),await e()}catch(v){k(v),p&&(p.disabled=!1,p.textContent="Add server")}}function k(d){let m=t.querySelector("#targets-error");m||(n.insertAdjacentHTML("afterbegin",'<p id="targets-error" class="error"></p>'),m=t.querySelector("#targets-error")),m.textContent=String(d instanceof Error?d.message:d)}function T(){var d;(d=t.querySelector("#targets-error"))==null||d.remove()}return()=>{a=!0}}function ee(t,a){const n=t.wire,e=t.mode==="local"?"this machine":"SSH",c=t.mode==="ssh"&&t.ssh?`${o(t.ssh.User)}@${o(t.ssh.Host)}`:e;let l,b;if(!n)l=x("not set up","neutral"),b=`<a class="btn" href="#/setup/${encodeURIComponent(t.id)}">Run setup wizard</a>`;else{const w=a.networks.find(k=>k.ChainID===n.ChainID),L=w?w.Name:`chain ${n.ChainID}`;l=`${x(L,"ok")} ${x(n.ExecID,"neutral")} ${x(n.BeaconID,"neutral")}${n.Archive?" "+x("archive","warn"):""}`,b=`
      <a class="btn" href="#/dash/${encodeURIComponent(t.id)}">Dashboard</a>
      <a class="btn" href="#/logs/${encodeURIComponent(t.id)}">Logs</a>
      <a class="btn btn-ghost" href="#/setup/${encodeURIComponent(t.id)}">Re-run setup</a>
    `}return`
    <div class="card">
      <h2>${o(t.id)}</h2>
      <p class="muted">${c}</p>
      <p>${l}</p>
      <div class="card-actions">
        ${b}
        <button class="btn btn-danger" data-action="delete-target" data-id="${o(t.id)}">Remove</button>
      </div>
    </div>
  `}function Le(){return`
    <form class="card" id="ssh-add-form" onsubmit="return false">
      <h3>Add server over SSH</h3>
      <label>
        Host
        <input id="ssh-host" type="text" placeholder="203.0.113.10" autocomplete="off" />
      </label>
      <label>
        User
        <input id="ssh-user" type="text" placeholder="root" autocomplete="off" />
      </label>
      <label>
        Private key path
        <input id="ssh-key" type="text" placeholder="/home/me/.ssh/id_ed25519" autocomplete="off" />
      </label>
      <label>
        Port <span class="muted">(optional, default 22)</span>
        <input id="ssh-port" type="text" inputmode="numeric" placeholder="22" autocomplete="off" />
      </label>
      <label>
        Target name <span class="muted">(optional, defaults to the host)</span>
        <input id="ssh-id" type="text" placeholder="my-node" autocomplete="off" />
      </label>
      <p class="muted small">
        The key never leaves this machine — only its path is stored, and the
        connection is dialed immediately so the host key can be pinned
        (trust-on-first-use) before it's saved.
      </p>
      <button class="btn" type="button" id="ssh-submit" data-action="add-ssh">Add server</button>
    </form>
  `}function Te(){const t=navigator.userAgentData,a=(t==null?void 0:t.platform)||navigator.platform||navigator.userAgent;return/mac|win/i.test(a)&&!/linux|android/i.test(a)}function Ce(){return Te()?`Setup requires a Linux target. This machine doesn't look like Linux — use "Add server over SSH" below to set up a remote Linux box instead.`:"The machine running valve-node. Setup only works on a Linux target."}function Re(t){return t.toLowerCase().replace(/[^a-z0-9]+/g,"-").replace(/^-+|-+$/g,"")||"target"}const F=[{id:"preflight",title:"Preflight checks"},{id:"toolchain",title:"Ensure git + build toolchains"},{id:"install-exec",title:"Install execution client"},{id:"install-beacon",title:"Install beacon client"},{id:"wire",title:"Write JWT secret and systemd units"},{id:"start",title:"Start execution and beacon services"},{id:"handshake",title:"Verify execution/beacon handshake"}],De=[369,943,1],te={369:"default",943:"practise here first"};function Ae(t,a){let n=!1;const e={targetId:a,step:"network",catalog:null,loadError:null,chainId:369,execId:null,beaconId:null,archive:!1,dataDir:"",jwtPath:"",starting:!1,startError:null,events:[],streamStop:null};t.innerHTML=`<h1>Setup: ${o(a)}</h1><div id="wizard-body"><p class="muted">Loading catalog…</p></div><div id="wizard-footer">${C()}</div>`;const c=t.querySelector("#wizard-body"),l=t.querySelector("#wizard-footer");B(t,(u,h)=>{r(u,h)}),b();async function b(){try{const[u,h]=await Promise.all([U(),O()]);if(n)return;e.catalog=u;const p=h.find(v=>v.id===a);p!=null&&p.wire&&(e.chainId=p.wire.ChainID,e.execId=p.wire.ExecID,e.beaconId=p.wire.BeaconID,e.archive=p.wire.Archive),w()}catch(u){if(n)return;e.loadError=String(u instanceof Error?u.message:u),w()}}function w(){if(e.loadError){c.innerHTML=`<p class="error">Failed to load: ${o(e.loadError)}</p>`;return}e.catalog&&(c.innerHTML=`
      ${A(e.step)}
      ${k()}
    `,L())}function L(){var h;const u=(h=e.catalog)==null?void 0:h.networks.find(p=>p.ChainID===e.chainId);l.innerHTML=u?C(u.Name,u.LearnURL):C()}function k(){switch(e.step){case"network":return T();case"clients":return d();case"mode":return y();case"review":return S();case"run":return i()}}function T(){const u=e.catalog;return`
      <section>
        <h2>1. Choose a network</h2>
        <div class="card-grid">${De.map(p=>{const v=u.networks.find(M=>M.ChainID===p);if(!v)return"";const I=e.chainId===p,E=te[p]?x(te[p],p===369?"ok":"warn"):"";return`
        <button class="card card-selectable ${I?"selected":""}" data-action="pick-network" data-chain-id="${p}" type="button">
          <h3>${o(v.Name)} <span class="muted">(chain ${p})</span></h3>
          ${E}
          <p class="muted small">Checkpoint sync from ${o(v.CheckpointURL)}</p>
        </button>
      `}).join("")}</div>
        <div class="wizard-actions">
          <button class="btn" data-action="goto-clients" ${e.chainId===null?"disabled":""}>Next: clients</button>
        </div>
      </section>
    `}function d(){const u=e.catalog,h=u.networks.find(I=>I.ChainID===e.chainId);if(!h)return'<p class="error">Unknown network.</p>';(e.execId===null||!h.ExecClients.includes(e.execId))&&(e.execId=h.ExecClients[0]??null),(e.beaconId===null||!h.BeaconClients.includes(e.beaconId))&&(e.beaconId=h.BeaconClients[0]??null);const p=h.ExecClients.map(I=>m(I,u,e.execId)).join(""),v=h.BeaconClients.map(I=>m(I,u,e.beaconId)).join("");return`
      <section>
        <h2>2. Choose your client pair</h2>
        <p class="muted">Only combinations known to work on ${o(h.Name)} are offered.</p>
        <label>
          Execution client
          <select id="exec-select" data-action="pick-exec">${p}</select>
        </label>
        <label>
          Beacon client
          <select id="beacon-select" data-action="pick-beacon">${v}</select>
        </label>
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-network">Back</button>
          <button class="btn" data-action="goto-mode">Next: mode</button>
        </div>
      </section>
    `}function m(u,h,p){const v=h.clients.find(E=>E.id===u),I=v?`${v.id} (${v.toolchain})`:u;return`<option value="${o(u)}" ${u===p?"selected":""}>${o(I)}</option>`}function y(){const u=e.chainId!==null?`/var/lib/valve-node/${e.chainId}`:"";return`
      <section>
        <h2>3. Choose sync mode</h2>
        <label class="radio">
          <input type="radio" name="mode" value="full" data-action="pick-mode" ${e.archive?"":"checked"} />
          Full — prune old state, smaller disk footprint
        </label>
        <label class="radio">
          <input type="radio" name="mode" value="archive" data-action="pick-mode" ${e.archive?"checked":""} />
          Archive — keep full history, needs much more disk
        </label>
        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            Data directory <span class="muted">(default: ${o(u)})</span>
            <input id="data-dir-input" type="text" placeholder="${o(u)}" value="${o(e.dataDir)}" />
          </label>
          <label>
            JWT secret path <span class="muted">(default: &lt;data dir&gt;/jwt.hex)</span>
            <input id="jwt-path-input" type="text" placeholder="${o(u)}/jwt.hex" value="${o(e.jwtPath)}" />
          </label>
        </details>
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-clients">Back</button>
          <button class="btn" data-action="goto-review">Next: review</button>
        </div>
      </section>
    `}function S(){const h=e.catalog.networks.find(E=>E.ChainID===e.chainId),p=e.dataDir||`/var/lib/valve-node/${e.chainId}`,v=e.jwtPath||`${p}/jwt.hex`,I=F.map(E=>`<li>${o(E.title)}</li>`).join("");return`
      <section>
        <h2>4. Review</h2>
        <table class="review-table">
          <tbody>
            <tr><th>Target</th><td>${o(e.targetId)}</td></tr>
            <tr><th>Network</th><td>${o((h==null?void 0:h.Name)??String(e.chainId))} (chain ${e.chainId})</td></tr>
            <tr><th>Execution client</th><td>${o(e.execId??"")}</td></tr>
            <tr><th>Beacon client</th><td>${o(e.beaconId??"")}</td></tr>
            <tr><th>Mode</th><td>${e.archive?"Archive":"Full"}</td></tr>
            <tr><th>Data directory</th><td><code>${o(p)}</code></td></tr>
            <tr><th>JWT secret path</th><td><code>${o(v)}</code></td></tr>
          </tbody>
        </table>
        <p class="muted small">
          There is no preview API for the exact files/units that will be
          written — the list below is the fixed step sequence setup always
          runs; the actual commands and file contents stream live once you
          start.
        </p>
        <ol class="step-preview">${I}</ol>
        ${e.startError?`<p class="error">${o(e.startError)}</p>`:""}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-mode">Back</button>
          <button class="btn btn-primary" data-action="start-setup" ${e.starting?"disabled":""}>
            ${e.starting?"Starting…":"Start setup"}
          </button>
        </div>
      </section>
    `}function i(){const h=e.catalog.networks.find(g=>g.ChainID===e.chainId),p=h==null?void 0:h.LearnURL,v=new Set(e.events.filter(g=>g.done).map(g=>g.stepId)),I=new Set(e.events.filter(g=>g.err).map(g=>g.stepId)),E=new Map;for(const g of e.events){if(!g.line)continue;const N=E.get(g.stepId)??[];N.push(g.line),E.set(g.stepId,N)}const M=F.map(g=>{var G;const N=v.has(g.id),_=I.has(g.id),re=_?x("failed","bad"):N?x("done","ok"):x("pending","neutral"),W=(E.get(g.id)??[]).slice(-5),K=(G=e.events.find(q=>q.stepId===g.id&&q.err))==null?void 0:G.err,se=g.id==="handshake"?`<p class="muted small">"Talking" means the beacon client can reach the execution client's Engine API over the shared JWT secret and both report the same head — the sign your node is wired correctly.${p?` <a href="${o(p)}" target="_blank" rel="noopener noreferrer">Learn more →</a>`:""}</p>`:"";return`
        <li class="step-row ${N?"step-done":""} ${_?"step-error":""}">
          <div class="step-head">${re} <strong>${o(g.title)}</strong></div>
          ${se}
          ${W.length?`<pre class="step-log">${W.map(q=>o(q)).join(`
`)}</pre>`:""}
          ${K?`<p class="error small">${o(K)}</p>`:""}
        </li>
      `}).join(""),J=e.events.some(g=>g.err),ae=F.every(g=>v.has(g.id))||e.events.some(g=>g.stepId==="handshake"&&g.done);return`
      <section>
        <h2>5. Running setup</h2>
        <ol class="step-list">${M}</ol>
        ${ae&&!J?`<p class="ok">Setup complete. <a href="#/dash/${encodeURIComponent(e.targetId)}">Open the dashboard →</a></p>`:""}
        ${J?'<button class="btn" data-action="start-setup">Retry setup</button>':""}
      </section>
    `}function r(u,h){switch(u){case"pick-network":e.chainId=Number(h.dataset.chainId),e.execId=null,e.beaconId=null,w();break;case"goto-network":e.step="network",w();break;case"goto-clients":if(e.chainId===null)return;e.step="clients",w();break;case"goto-mode":s(),e.step="mode",w();break;case"goto-review":f(),e.step="review",w();break;case"start-setup":$();break}}function s(){const u=t.querySelector("#exec-select"),h=t.querySelector("#beacon-select");u&&(e.execId=u.value),h&&(e.beaconId=h.value)}function f(){const u=t.querySelectorAll('input[name="mode"]');for(const v of Array.from(u))v.checked&&(e.archive=v.value==="archive");const h=t.querySelector("#data-dir-input"),p=t.querySelector("#jwt-path-input");h&&(e.dataDir=h.value.trim()),p&&(e.jwtPath=p.value.trim())}async function $(){var h;if(e.chainId===null||!e.execId||!e.beaconId)return;e.starting=!0,e.startError=null,w();const u={ChainID:e.chainId,ExecID:e.execId,BeaconID:e.beaconId,Archive:e.archive};e.dataDir&&(u.DataDir=e.dataDir),e.jwtPath&&(u.JWTPath=e.jwtPath);try{await le(e.targetId,u)}catch(p){if(!(p instanceof z&&p.status===409)){e.starting=!1,e.startError=String(p instanceof Error?p.message:p),w();return}}e.starting=!1,e.step="run",e.events=[],w(),(h=e.streamStop)==null||h.call(e),e.streamStop=de(e.targetId,p=>{n||(e.events.push(p),e.step==="run"&&w())})}function A(u){const h=[{id:"network",label:"Network"},{id:"clients",label:"Clients"},{id:"mode",label:"Mode"},{id:"review",label:"Review"},{id:"run",label:"Run"}],v=h.map(I=>I.id).indexOf(u);return`
      <ol class="wizard-progress">
        ${h.map((I,E)=>`<li class="${E===v?"current":E<v?"past":"future"}">${o(I.label)}</li>`).join("")}
      </ol>
    `}return()=>{var u;n=!0,(u=e.streamStop)==null||u.call(e)}}const He=document.querySelector("#app"),{contentEl:Ne,setActiveNav:Pe}=ge(He);let D=null;function Me(){const a=location.hash.replace(/^#\/?/,"").split("/").filter(Boolean);if(a.length===0)return{screen:"targets"};const[n,e]=a;return n==="setup"||n==="dash"||n==="logs"?{screen:n,id:e?decodeURIComponent(e):void 0}:{screen:n??"targets"}}function P(t){const a=document.createElement("div");return Ne.replaceChildren(a),t(a)}function ne(){if(D){try{D()}catch{}D=null}const{screen:t,id:a}=Me();switch(Pe(t),t){case"setup":if(!a){location.hash="#/targets";return}D=P(n=>Ae(n,a));break;case"dash":if(!a){location.hash="#/targets";return}D=P(n=>we(n,a));break;case"logs":if(!a){location.hash="#/targets";return}D=P(n=>Se(n,a));break;case"settings":D=P(n=>xe(n));break;case"targets":default:D=P(n=>Ee(n));break}}window.addEventListener("hashchange",ne);ne();
