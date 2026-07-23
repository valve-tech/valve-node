var re=Object.defineProperty;var se=(t,n,r)=>n in t?re(t,n,{enumerable:!0,configurable:!0,writable:!0,value:r}):t[n]=r;var W=(t,n,r)=>se(t,typeof n!="symbol"?n+"":n,r);(function(){const n=document.createElement("link").relList;if(n&&n.supports&&n.supports("modulepreload"))return;for(const i of document.querySelectorAll('link[rel="modulepreload"]'))e(i);new MutationObserver(i=>{for(const c of i)if(c.type==="childList")for(const f of c.addedNodes)f.tagName==="LINK"&&f.rel==="modulepreload"&&e(f)}).observe(document,{childList:!0,subtree:!0});function r(i){const c={};return i.integrity&&(c.integrity=i.integrity),i.referrerPolicy&&(c.referrerPolicy=i.referrerPolicy),i.crossOrigin==="use-credentials"?c.credentials="include":i.crossOrigin==="anonymous"?c.credentials="omit":c.credentials="same-origin",c}function e(i){if(i.ep)return;i.ep=!0;const c=r(i);fetch(i.href,c)}})();function Z(){return T("/api/catalog")}function M(){return T("/api/targets")}function K(t){return T("/api/targets",{method:"POST",headers:q,body:JSON.stringify(t)})}function oe(t){return T(`/api/targets/${encodeURIComponent(t)}`,{method:"DELETE"})}function ie(t,n){return T(`/api/targets/${encodeURIComponent(t)}/setup`,{method:"POST",headers:q,body:JSON.stringify(n)})}function ce(t,n){const r=new EventSource(`/api/targets/${encodeURIComponent(t)}/setup/stream`);return r.onmessage=e=>{try{n(JSON.parse(e.data))}catch{}},()=>r.close()}function le(t,n){const r=new EventSource(`/api/targets/${encodeURIComponent(t)}/monitor/stream`);return r.onmessage=e=>{try{n(JSON.parse(e.data))}catch{}},()=>r.close()}function de(t,n=200){return T(`/api/targets/${encodeURIComponent(t)}/logs?n=${n}`)}function ue(t,n){const r=new EventSource(`/api/targets/${encodeURIComponent(t)}/logs/stream`);return r.onmessage=e=>{try{n(JSON.parse(e.data))}catch{}},()=>r.close()}function G(t,n){const r=n===void 0?{}:{lines:n};return T(`/api/targets/${encodeURIComponent(t)}/explain`,{method:"POST",headers:q,body:JSON.stringify(r)})}function pe(){return T("/api/settings")}function he(t){return T("/api/settings",{method:"PUT",headers:q,body:JSON.stringify(t)})}class j extends Error{constructor(r,e){super(e);W(this,"status");this.name="ApiError",this.status=r}}const q={"Content-Type":"application/json"};async function T(t,n){const r=await fetch(t,n);if(!r.ok){let i=r.statusText||`HTTP ${r.status}`;try{const c=await r.json();c&&typeof c.error=="string"&&c.error&&(i=c.error)}catch{}throw new j(r.status,i)}if(r.status===204)return;const e=await r.text();return e?JSON.parse(e):void 0}const fe="https://learn.valve.city/rpc";function o(t){return t.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;").replace(/'/g,"&#39;")}function H(t,n){return`
    <footer class="footer">
      <a href="${o(fe)}" target="_blank" rel="noopener noreferrer">Learn how this works → learn.valve.city/rpc</a>
    </footer>
  `}function me(t){t.innerHTML=`
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
  `;const n=t.querySelector("#content"),r=Array.from(t.querySelectorAll("[data-nav]"));return{contentEl:n,setActiveNav:i=>{for(const c of r)c.classList.toggle("active",c.dataset.nav===i)}}}function R(t){return Number.isFinite(t)?t.toLocaleString("en-US"):"—"}function ve(t){return Number.isFinite(t)?`${t.toFixed(1)}%`:"—"}function ge(t){if(!Number.isFinite(t)||t<0)return"—";if(t<60)return`~${Math.round(t)}s`;const n=Math.round(t/60),r=Math.floor(n/60),e=n%60;if(r===0)return`~${e}m`;if(r<48)return`~${r}h ${e}m`;const i=Math.floor(r/24),c=r%24;return`~${i}d ${c}h`}function k(t,n){return`<span class="badge badge-${n}">${o(t)}</span>`}function O(t,n){t.addEventListener("click",r=>{const e=r.target.closest("[data-action]");if(!e||!t.contains(e))return;const i=e.dataset.action;i&&n(i,e,r)})}const be=85;function ye(t,n){let r=!1,e=null,i=null,c=null;t.innerHTML=`<h1>Dashboard: ${o(n)}</h1><div id="dash-body"><p class="muted">Loading…</p></div>${H()}`;const f=t.querySelector("#dash-body");x();async function x(){let a;try{a=(await M()).find(b=>b.id===n)}catch(s){if(r)return;f.innerHTML=`<p class="error">Failed to load target: ${o(String(s))}</p>`;return}if(!r){if(!a){f.innerHTML=`<p class="error">Target "${o(n)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!a.wire){f.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(n)}">Run the setup wizard →</a></p>`;return}f.innerHTML='<p class="muted">Connecting…</p>',e=le(n,s=>{r||(E(s),S(s),i=s)})}}function E(a){if(!i)return;const s=(new Date(a.at).getTime()-new Date(i.at).getTime())/1e3,b=a.execHead-i.execHead;if(s>0&&b>=0){const I=b/s;c=c===null?I:c*.7+I*.3}}function S(a){f.innerHTML=`
      <div class="card-grid">
        ${L(a)}
        ${d(a)}
        ${m(a)}
        ${v(a)}
        ${l(a)}
      </div>
      <p class="muted small">Last updated ${o(new Date(a.at).toLocaleTimeString())}</p>
    `}function L(a){const s=a.refHead>0,b=s?a.refHead-a.execHead:null,I=b!==null&&b>0&&c&&c>0?ge(b/c):b!==null&&b<=0?"caught up":"—";return`
      <div class="card">
        <h3>Execution sync</h3>
        <p>${a.execSyncing?k("syncing","warn"):k("synced","ok")}</p>
        <dl class="stat-list">
          <div><dt>Local head</dt><dd>${R(a.execHead)}</dd></div>
          <div><dt>Reference head</dt><dd>${s?R(a.refHead):"unavailable"}</dd></div>
          <div><dt>Lag</dt><dd>${b!==null?R(Math.max(b,0))+" blocks":"—"}</dd></div>
          <div><dt>ETA</dt><dd>${I}</dd></div>
        </dl>
      </div>
    `}function d(a){return`
      <div class="card">
        <h3>Beacon sync</h3>
        <p>${a.beaconDistance===0?k("synced","ok"):k("syncing","warn")}</p>
        <dl class="stat-list">
          <div><dt>Slot</dt><dd>${R(a.beaconSlot)}</dd></div>
          <div><dt>Distance</dt><dd>${R(a.beaconDistance)}</dd></div>
        </dl>
      </div>
    `}function m(a){return`
      <div class="card">
        <h3>Peers</h3>
        <dl class="stat-list">
          <div><dt>Execution</dt><dd>${R(a.execPeers)}</dd></div>
          <div><dt>Beacon</dt><dd>${R(a.beaconPeers)}</dd></div>
        </dl>
      </div>
    `}function v(a){const s=a.diskUsedPct>=be;return`
      <div class="card ${s?"card-warn":""}">
        <h3>Disk</h3>
        <div class="meter"><div class="meter-fill ${s?"meter-warn":""}" style="width:${Math.min(a.diskUsedPct,100)}%"></div></div>
        <p>${ve(a.diskUsedPct)} used</p>
      </div>
    `}function l(a){return`
      <div class="card">
        <h3>Services</h3>
        <p>Execution ${a.execActive?k("active","ok"):k("down","bad")}</p>
        <p>Beacon ${a.beaconActive?k("active","ok"):k("down","bad")}</p>
        <p><a href="#/logs/${encodeURIComponent(n)}">View logs →</a></p>
      </div>
    `}return()=>{r=!0,e==null||e()}}const V=500,Y="valve-node.explain-consent";function $e(t,n){let r=!1,e=null;const i=[];t.innerHTML=`
    <h1>Logs: ${o(n)}</h1>
    <div id="logs-body"><p class="muted">Loading…</p></div>
    ${H()}
  `;const c=t.querySelector("#logs-body");O(t,l=>{l==="explain"&&S()}),f();async function f(){let l;try{l=(await M()).find(s=>s.id===n)}catch(a){if(r)return;c.innerHTML=`<p class="error">Failed to load target: ${o(String(a))}</p>`;return}if(!r){if(!l){c.innerHTML=`<p class="error">Target "${o(n)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!l.wire){c.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(n)}">Run the setup wizard →</a></p>`;return}try{const a=await de(n,200);if(r)return;i.push(...a)}catch(a){if(r)return;c.innerHTML=`<p class="error">Failed to load logs: ${o(String(a))}</p>`;return}x(),e=ue(n,a=>{r||(i.push(a),i.length>V&&i.splice(0,i.length-V),x())})}}function x(){const l=i.filter(s=>s.severity==="error"||s.severity==="critical");c.innerHTML=`
      <div class="logs-layout">
        <section class="logs-tail">
          <div class="logs-tail-head">
            <h2>Live tail</h2>
            <button class="btn" data-action="explain">Explain with AI</button>
          </div>
          <div class="log-lines">${i.map(E).join("")}</div>
        </section>
        <section class="logs-errors">
          <h2>Error feed ${k(String(l.length),l.length?"bad":"neutral")}</h2>
          <div class="log-lines">${l.length?l.slice().reverse().map(E).join(""):'<p class="muted">No errors seen yet.</p>'}</div>
        </section>
      </div>
    `;const a=c.querySelector(".log-lines");a&&(a.scrollTop=a.scrollHeight)}function E(l){const a=l.severity||"info",s=l.learnUrl?` <a href="${o(l.learnUrl)}" target="_blank" rel="noopener noreferrer">learn →</a>`:"";return`
      <div class="log-line log-${o(a)}">
        <span class="log-time">${o(new Date(l.at).toLocaleTimeString())}</span>
        <span class="log-unit">${o(l.unit)}</span>
        <span class="log-sev">${o(a)}</span>
        <span class="log-text">${o(l.line)}</span>
        ${l.explain?`<div class="log-explain">${o(l.explain)}${s}</div>`:""}
      </div>
    `}async function S(){const l=i.filter(s=>s.severity==="error"||s.severity==="critical").map(s=>s.line).slice(-40);if(!(localStorage.getItem(Y)==="1")){L(l);return}await d(l)}function L(l){const a=l.length?`<pre class="explain-excerpt">${l.map(s=>o(s)).join(`
`)}</pre>`:'<p class="muted">No recent error lines are loaded yet — the server will auto-select its own recent error/critical lines instead.</p>';m(`
      <h2>Send logs to your AI provider?</h2>
      <p>
        The excerpt below will be sent to the AI provider configured in
        <a href="#/settings">Settings</a> to generate a plain-English
        explanation. This happens every time you click "Explain with AI";
        this confirmation only shows once per browser.
      </p>
      ${a}
      <div class="modal-actions">
        <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        <button class="btn btn-primary" data-modal-action="proceed">Send to AI provider</button>
      </div>
    `,s=>{s==="proceed"?(localStorage.setItem(Y,"1"),v(),d(l)):v()})}async function d(l){m('<h2>Explain with AI</h2><p class="muted">Asking the AI provider…</p>',()=>{});try{const a=l.length?await G(n,l):await G(n);if(r)return;m(`
        <h2>Explanation</h2>
        <div class="explain-text">${o(a.text)}</div>
        <details class="advanced">
          <summary>What was sent</summary>
          <pre class="explain-excerpt">${a.sentExcerpt.map(s=>o(s)).join(`
`)||"(no log lines — general question only)"}</pre>
        </details>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,s=>{s==="close"&&v()})}catch(a){if(r)return;if(a instanceof j&&a.status===409){m(`
          <h2>No AI provider configured</h2>
          <p>Set a provider and key in <a href="#/settings">Settings</a>, then try again.</p>
          <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
        `,s=>{s==="close"&&v()});return}m(`
        <h2>Explain failed</h2>
        <p class="error">${o(a instanceof Error?a.message:String(a))}</p>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,s=>{s==="close"&&v()})}}function m(l,a){v();const s=document.createElement("div");s.className="modal-overlay",s.id="explain-modal",s.innerHTML=`<div class="modal">${l}</div>`,s.addEventListener("click",b=>{const I=b.target.closest("[data-modal-action]");I!=null&&I.dataset.modalAction&&a(I.dataset.modalAction),b.target===s&&a("cancel")}),document.body.appendChild(s)}function v(){var l;(l=document.getElementById("explain-modal"))==null||l.remove()}return()=>{r=!0,e==null||e(),v()}}const we=[{value:"",label:"None"},{value:"gemini",label:"Gemini"},{value:"groq",label:"Groq"},{value:"ollama",label:"Ollama"}];function Se(t){let n=!1,r=!1,e=!1,i=null,c=!1,f=null;t.innerHTML=`<h1>Settings</h1><div id="settings-body"><p class="muted">Loading…</p></div>${H()}`;const x=t.querySelector("#settings-body");O(t,d=>{if(d==="save"&&L(),d==="clear-key"){if(!f)return;r=!0;const m=t.querySelector("#ai-key");m&&(m.value=""),S(f)}}),E();async function E(){try{const d=await pe();if(n)return;f=d,S(d)}catch(d){if(n)return;x.innerHTML=`<p class="error">Failed to load settings: ${o(String(d))}</p>`}}function S(d){var l,a;const m=we.map(s=>`<option value="${s.value}" ${d.aiProvider===s.value?"selected":""}>${o(s.label)}</option>`).join("");x.innerHTML=`
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
        ${i?`<p class="error">${o(i)}</p>`:""}
        ${c?'<p class="ok">Saved.</p>':""}
        <button class="btn btn-primary" type="button" data-action="save" ${e?"disabled":""}>${e?"Saving…":"Save"}</button>
      </form>
    `;const v=t.querySelector("#ai-key");v==null||v.addEventListener("input",()=>{r=!0,c=!1}),(l=t.querySelector("#ai-provider"))==null||l.addEventListener("change",()=>{c=!1}),(a=t.querySelector("#ref-rpc-base"))==null||a.addEventListener("input",()=>{c=!1})}async function L(){const d=t.querySelector("#ai-provider"),m=t.querySelector("#ai-key"),v=t.querySelector("#ref-rpc-base");if(!d||!m||!v||!f)return;const l={aiProvider:d.value,refRpcBase:v.value.trim()};r&&(l.aiKey=m.value),e=!0,i=null,c=!1,S(f);try{const a=await he(l);if(n)return;f=a,r=!1,e=!1,c=!0,S(a)}catch(a){if(n)return;e=!1,i=String(a instanceof Error?a.message:a),S(f)}}return()=>{n=!0}}const ke="local";function xe(t){let n=!1;t.innerHTML=`
    <h1>Targets</h1>
    <div id="targets-body"><p class="muted">Loading…</p></div>
    ${H()}
  `;const r=t.querySelector("#targets-body");O(t,(d,m)=>{c(d,m)}),e();async function e(){try{const[d,m]=await Promise.all([M(),Z()]);if(n)return;i(d,m)}catch(d){if(n)return;r.innerHTML=`<p class="error">Failed to load targets: ${o(String(d))}</p>`}}function i(d,m){const v=d.find(b=>b.mode==="local"),l=d.filter(b=>b.mode==="ssh"),a=v?Q(v,m):`
        <div class="card">
          <h2>This machine</h2>
          <p class="muted">${Le()}</p>
          <button class="btn" data-action="add-local">Add this machine as a target</button>
        </div>
      `,s=l.length?l.map(b=>Q(b,m)).join(""):'<p class="muted">No SSH targets yet.</p>';r.innerHTML=`
      <section class="section">${a}</section>
      <section class="section">
        <h2>Servers over SSH</h2>
        <div class="card-grid">${s}</div>
        ${Ie()}
      </section>
    `}async function c(d,m){if(d==="add-local"){await f();return}if(d==="delete-target"){const v=m.dataset.id;if(!v||!confirm(`Remove target "${v}"? This does not touch anything already running on it.`))return;await x(v);return}d==="add-ssh"&&await E()}async function f(){L();try{await K({id:ke,mode:"local"}),await e()}catch(d){S(d)}}async function x(d){try{await oe(d),await e()}catch(m){S(m)}}async function E(){const d=t.querySelector("#ssh-host"),m=t.querySelector("#ssh-user"),v=t.querySelector("#ssh-key"),l=t.querySelector("#ssh-port"),a=t.querySelector("#ssh-id");if(!d||!m||!v||!l||!a)return;const s=d.value.trim(),b=m.value.trim(),I=v.value.trim(),u=l.value.trim(),p=a.value.trim();if(L(),!s||!b||!I){S(new Error("host, user, and key path are required"));return}const h=p||Te(s),y={Host:s,User:b,KeyPath:I};if(u){const w=Number.parseInt(u,10);if(!Number.isFinite(w)||w<=0){S(new Error("port must be a positive number"));return}y.Port=w}const $=t.querySelector("#ssh-submit");$&&($.disabled=!0,$.textContent="Connecting…");try{await K({id:h,mode:"ssh",ssh:y}),await e()}catch(w){S(w),$&&($.disabled=!1,$.textContent="Add server")}}function S(d){let m=t.querySelector("#targets-error");m||(r.insertAdjacentHTML("afterbegin",'<p id="targets-error" class="error"></p>'),m=t.querySelector("#targets-error")),m.textContent=String(d instanceof Error?d.message:d)}function L(){var d;(d=t.querySelector("#targets-error"))==null||d.remove()}return()=>{n=!0}}function Q(t,n){const r=t.wire,e=t.mode==="local"?"this machine":"SSH",i=t.mode==="ssh"&&t.ssh?`${o(t.ssh.User)}@${o(t.ssh.Host)}`:e;let c,f;if(!r)c=k("not set up","neutral"),f=`<a class="btn" href="#/setup/${encodeURIComponent(t.id)}">Run setup wizard</a>`;else{const x=n.networks.find(S=>S.ChainID===r.ChainID),E=x?x.Name:`chain ${r.ChainID}`;c=`${k(E,"ok")} ${k(r.ExecID,"neutral")} ${k(r.BeaconID,"neutral")}${r.Archive?" "+k("archive","warn"):""}`,f=`
      <a class="btn" href="#/dash/${encodeURIComponent(t.id)}">Dashboard</a>
      <a class="btn" href="#/logs/${encodeURIComponent(t.id)}">Logs</a>
      <a class="btn btn-ghost" href="#/setup/${encodeURIComponent(t.id)}">Re-run setup</a>
    `}return`
    <div class="card">
      <h2>${o(t.id)}</h2>
      <p class="muted">${i}</p>
      <p>${c}</p>
      <div class="card-actions">
        ${f}
        <button class="btn btn-danger" data-action="delete-target" data-id="${o(t.id)}">Remove</button>
      </div>
    </div>
  `}function Ie(){return`
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
  `}function Ee(){const t=navigator.userAgentData,n=(t==null?void 0:t.platform)||navigator.platform||navigator.userAgent;return/mac|win/i.test(n)&&!/linux|android/i.test(n)}function Le(){return Ee()?`Setup requires a Linux target. This machine doesn't look like Linux — use "Add server over SSH" below to set up a remote Linux box instead.`:"The machine running valve-node. Setup only works on a Linux target."}function Te(t){return t.toLowerCase().replace(/[^a-z0-9]+/g,"-").replace(/^-+|-+$/g,"")||"target"}const U=[{id:"preflight",title:"Preflight checks"},{id:"toolchain",title:"Ensure git + build toolchains"},{id:"install-exec",title:"Install execution client"},{id:"install-beacon",title:"Install beacon client"},{id:"wire",title:"Write JWT secret and systemd units"},{id:"start",title:"Start execution and beacon services"},{id:"handshake",title:"Verify execution/beacon handshake"}],Ce=[369,943,1],X={369:"default",943:"practise here first"};function Re(t,n){let r=!1;const e={targetId:n,step:"network",catalog:null,loadError:null,chainId:369,execId:null,beaconId:null,archive:!1,dataDir:"",jwtPath:"",starting:!1,startError:null,events:[],streamStop:null};t.innerHTML=`<h1>Setup: ${o(n)}</h1><div id="wizard-body"><p class="muted">Loading catalog…</p></div>${H()}`;const i=t.querySelector("#wizard-body");O(t,(u,p)=>{l(u,p)}),c();async function c(){try{const[u,p]=await Promise.all([Z(),M()]);if(r)return;e.catalog=u;const h=p.find(y=>y.id===n);h!=null&&h.wire&&(e.chainId=h.wire.ChainID,e.execId=h.wire.ExecID,e.beaconId=h.wire.BeaconID,e.archive=h.wire.Archive),f()}catch(u){if(r)return;e.loadError=String(u instanceof Error?u.message:u),f()}}function f(){if(e.loadError){i.innerHTML=`<p class="error">Failed to load: ${o(e.loadError)}</p>`;return}e.catalog&&(i.innerHTML=`
      ${I(e.step)}
      ${x()}
    `)}function x(){switch(e.step){case"network":return E();case"clients":return S();case"mode":return d();case"review":return m();case"run":return v()}}function E(){const u=e.catalog;return`
      <section>
        <h2>1. Choose a network</h2>
        <div class="card-grid">${Ce.map(h=>{const y=u.networks.find(N=>N.ChainID===h);if(!y)return"";const $=e.chainId===h,w=X[h]?k(X[h],h===369?"ok":"warn"):"";return`
        <button class="card card-selectable ${$?"selected":""}" data-action="pick-network" data-chain-id="${h}" type="button">
          <h3>${o(y.Name)} <span class="muted">(chain ${h})</span></h3>
          ${w}
          <p class="muted small">Checkpoint sync from ${o(y.CheckpointURL)}</p>
        </button>
      `}).join("")}</div>
        <div class="wizard-actions">
          <button class="btn" data-action="goto-clients" ${e.chainId===null?"disabled":""}>Next: clients</button>
        </div>
      </section>
    `}function S(){const u=e.catalog,p=u.networks.find($=>$.ChainID===e.chainId);if(!p)return'<p class="error">Unknown network.</p>';(e.execId===null||!p.ExecClients.includes(e.execId))&&(e.execId=p.ExecClients[0]??null),(e.beaconId===null||!p.BeaconClients.includes(e.beaconId))&&(e.beaconId=p.BeaconClients[0]??null);const h=p.ExecClients.map($=>L($,u,e.execId)).join(""),y=p.BeaconClients.map($=>L($,u,e.beaconId)).join("");return`
      <section>
        <h2>2. Choose your client pair</h2>
        <p class="muted">Only combinations known to work on ${o(p.Name)} are offered.</p>
        <label>
          Execution client
          <select id="exec-select" data-action="pick-exec">${h}</select>
        </label>
        <label>
          Beacon client
          <select id="beacon-select" data-action="pick-beacon">${y}</select>
        </label>
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-network">Back</button>
          <button class="btn" data-action="goto-mode">Next: mode</button>
        </div>
      </section>
    `}function L(u,p,h){const y=p.clients.find(w=>w.id===u),$=y?`${y.id} (${y.toolchain})`:u;return`<option value="${o(u)}" ${u===h?"selected":""}>${o($)}</option>`}function d(){const u=e.chainId!==null?`/var/lib/valve-node/${e.chainId}`:"";return`
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
    `}function m(){const p=e.catalog.networks.find(w=>w.ChainID===e.chainId),h=e.dataDir||`/var/lib/valve-node/${e.chainId}`,y=e.jwtPath||`${h}/jwt.hex`,$=U.map(w=>`<li>${o(w.title)}</li>`).join("");return`
      <section>
        <h2>4. Review</h2>
        <table class="review-table">
          <tbody>
            <tr><th>Target</th><td>${o(e.targetId)}</td></tr>
            <tr><th>Network</th><td>${o((p==null?void 0:p.Name)??String(e.chainId))} (chain ${e.chainId})</td></tr>
            <tr><th>Execution client</th><td>${o(e.execId??"")}</td></tr>
            <tr><th>Beacon client</th><td>${o(e.beaconId??"")}</td></tr>
            <tr><th>Mode</th><td>${e.archive?"Archive":"Full"}</td></tr>
            <tr><th>Data directory</th><td><code>${o(h)}</code></td></tr>
            <tr><th>JWT secret path</th><td><code>${o(y)}</code></td></tr>
          </tbody>
        </table>
        <p class="muted small">
          There is no preview API for the exact files/units that will be
          written — the list below is the fixed step sequence setup always
          runs; the actual commands and file contents stream live once you
          start.
        </p>
        <ol class="step-preview">${$}</ol>
        ${e.startError?`<p class="error">${o(e.startError)}</p>`:""}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-mode">Back</button>
          <button class="btn btn-primary" data-action="start-setup" ${e.starting?"disabled":""}>
            ${e.starting?"Starting…":"Start setup"}
          </button>
        </div>
      </section>
    `}function v(){const p=e.catalog.networks.find(g=>g.ChainID===e.chainId),h=p==null?void 0:p.LearnURL,y=new Set(e.events.filter(g=>g.done).map(g=>g.stepId)),$=new Set(e.events.filter(g=>g.err).map(g=>g.stepId)),w=new Map;for(const g of e.events){if(!g.line)continue;const A=w.get(g.stepId)??[];A.push(g.line),w.set(g.stepId,A)}const N=U.map(g=>{var _;const A=y.has(g.id),F=$.has(g.id),ne=F?k("failed","bad"):A?k("done","ok"):k("pending","neutral"),J=(w.get(g.id)??[]).slice(-5),z=(_=e.events.find(P=>P.stepId===g.id&&P.err))==null?void 0:_.err,ae=g.id==="handshake"?`<p class="muted small">"Talking" means the beacon client can reach the execution client's Engine API over the shared JWT secret and both report the same head — the sign your node is wired correctly.${h?` <a href="${o(h)}" target="_blank" rel="noopener noreferrer">Learn more →</a>`:""}</p>`:"";return`
        <li class="step-row ${A?"step-done":""} ${F?"step-error":""}">
          <div class="step-head">${ne} <strong>${o(g.title)}</strong></div>
          ${ae}
          ${J.length?`<pre class="step-log">${J.map(P=>o(P)).join(`
`)}</pre>`:""}
          ${z?`<p class="error small">${o(z)}</p>`:""}
        </li>
      `}).join(""),B=e.events.some(g=>g.err),te=U.every(g=>y.has(g.id))||e.events.some(g=>g.stepId==="handshake"&&g.done);return`
      <section>
        <h2>5. Running setup</h2>
        <ol class="step-list">${N}</ol>
        ${te&&!B?`<p class="ok">Setup complete. <a href="#/dash/${encodeURIComponent(e.targetId)}">Open the dashboard →</a></p>`:""}
        ${B?'<button class="btn" data-action="start-setup">Retry setup</button>':""}
      </section>
    `}function l(u,p){switch(u){case"pick-network":e.chainId=Number(p.dataset.chainId),e.execId=null,e.beaconId=null,f();break;case"goto-network":e.step="network",f();break;case"goto-clients":if(e.chainId===null)return;e.step="clients",f();break;case"goto-mode":a(),e.step="mode",f();break;case"goto-review":s(),e.step="review",f();break;case"start-setup":b();break}}function a(){const u=t.querySelector("#exec-select"),p=t.querySelector("#beacon-select");u&&(e.execId=u.value),p&&(e.beaconId=p.value)}function s(){const u=t.querySelectorAll('input[name="mode"]');for(const y of Array.from(u))y.checked&&(e.archive=y.value==="archive");const p=t.querySelector("#data-dir-input"),h=t.querySelector("#jwt-path-input");p&&(e.dataDir=p.value.trim()),h&&(e.jwtPath=h.value.trim())}async function b(){var p;if(e.chainId===null||!e.execId||!e.beaconId)return;e.starting=!0,e.startError=null,f();const u={ChainID:e.chainId,ExecID:e.execId,BeaconID:e.beaconId,Archive:e.archive};e.dataDir&&(u.DataDir=e.dataDir),e.jwtPath&&(u.JWTPath=e.jwtPath);try{await ie(e.targetId,u)}catch(h){if(!(h instanceof j&&h.status===409)){e.starting=!1,e.startError=String(h instanceof Error?h.message:h),f();return}}e.starting=!1,e.step="run",e.events=[],f(),(p=e.streamStop)==null||p.call(e),e.streamStop=ce(e.targetId,h=>{r||(e.events.push(h),e.step==="run"&&f())})}function I(u){const p=[{id:"network",label:"Network"},{id:"clients",label:"Clients"},{id:"mode",label:"Mode"},{id:"review",label:"Review"},{id:"run",label:"Run"}],y=p.map($=>$.id).indexOf(u);return`
      <ol class="wizard-progress">
        ${p.map(($,w)=>`<li class="${w===y?"current":w<y?"past":"future"}">${o($.label)}</li>`).join("")}
      </ol>
    `}return()=>{var u;r=!0,(u=e.streamStop)==null||u.call(e)}}const Ae=document.querySelector("#app"),{contentEl:D,setActiveNav:De}=me(Ae);let C=null;function He(){const n=location.hash.replace(/^#\/?/,"").split("/").filter(Boolean);if(n.length===0)return{screen:"targets"};const[r,e]=n;return r==="setup"||r==="dash"||r==="logs"?{screen:r,id:e?decodeURIComponent(e):void 0}:{screen:r??"targets"}}function ee(){if(C){try{C()}catch{}C=null}const{screen:t,id:n}=He();switch(De(t),t){case"setup":if(!n){location.hash="#/targets";return}C=Re(D,n);break;case"dash":if(!n){location.hash="#/targets";return}C=ye(D,n);break;case"logs":if(!n){location.hash="#/targets";return}C=$e(D,n);break;case"settings":C=Se(D);break;case"targets":default:C=xe(D);break}}window.addEventListener("hashchange",ee);ee();
