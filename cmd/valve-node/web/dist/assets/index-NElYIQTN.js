var ye=Object.defineProperty;var $e=(t,n,a)=>n in t?ye(t,n,{enumerable:!0,configurable:!0,writable:!0,value:a}):t[n]=a;var se=(t,n,a)=>$e(t,typeof n!="symbol"?n+"":n,a);(function(){const n=document.createElement("link").relList;if(n&&n.supports&&n.supports("modulepreload"))return;for(const l of document.querySelectorAll('link[rel="modulepreload"]'))e(l);new MutationObserver(l=>{for(const p of l)if(p.type==="childList")for(const T of p.addedNodes)T.tagName==="LINK"&&T.rel==="modulepreload"&&e(T)}).observe(document,{childList:!0,subtree:!0});function a(l){const p={};return l.integrity&&(p.integrity=l.integrity),l.referrerPolicy&&(p.referrerPolicy=l.referrerPolicy),l.crossOrigin==="use-credentials"?p.credentials="include":l.crossOrigin==="anonymous"?p.credentials="omit":p.credentials="same-origin",p}function e(l){if(l.ep)return;l.ep=!0;const p=a(l);fetch(l.href,p)}})();function X(){return A("/api/catalog")}function Y(){return A("/api/targets")}function ie(t){return A("/api/targets",{method:"POST",headers:Q,body:JSON.stringify(t)})}function we(t){return A(`/api/targets/${encodeURIComponent(t)}`,{method:"DELETE"})}function Te(t,n){return A(`/api/targets/${encodeURIComponent(t)}/setup`,{method:"POST",headers:Q,body:JSON.stringify(n)})}function Pe(t,n){const a=new EventSource(`/api/targets/${encodeURIComponent(t)}/setup/stream`);return a.onmessage=e=>{try{n(JSON.parse(e.data))}catch{}},()=>a.close()}function xe(t,n){const a=new EventSource(`/api/targets/${encodeURIComponent(t)}/monitor/stream`);return a.onmessage=e=>{try{n(JSON.parse(e.data))}catch{}},()=>a.close()}function Se(t,n=200){return A(`/api/targets/${encodeURIComponent(t)}/logs?n=${n}`)}function Ee(t,n){const a=new EventSource(`/api/targets/${encodeURIComponent(t)}/logs/stream`);return a.onmessage=e=>{try{n(JSON.parse(e.data))}catch{}},()=>a.close()}function ce(t,n){const a=n===void 0?{}:{lines:n};return A(`/api/targets/${encodeURIComponent(t)}/explain`,{method:"POST",headers:Q,body:JSON.stringify(a)})}function ke(t,n,a){return A(`/api/targets/${encodeURIComponent(t)}/services/${n}/${a}`,{method:"POST"})}function Ce(t,n){return A(`/api/targets/${encodeURIComponent(t)}/services/${n}/clear`,{method:"POST",headers:Q,body:JSON.stringify({Confirm:n})})}function Ie(t){return A(`/api/targets/${encodeURIComponent(t)}/du`)}function Le(t){return A(`/api/targets/${encodeURIComponent(t)}/endpoints`)}function He(t){return A(`/api/targets/${encodeURIComponent(t)}/firewall`)}function Re(){return A("/api/settings")}function Be(t){return A("/api/settings",{method:"PUT",headers:Q,body:JSON.stringify(t)})}class oe extends Error{constructor(a,e){super(e);se(this,"status");this.name="ApiError",this.status=a}}const Q={"Content-Type":"application/json"};async function A(t,n){const a=await fetch(t,n);if(!a.ok){let l=a.statusText||`HTTP ${a.status}`;try{const p=await a.json();p&&typeof p.error=="string"&&p.error&&(l=p.error)}catch{}throw new oe(a.status,l)}if(a.status===204)return;const e=await a.text();return e?JSON.parse(e):void 0}const De="https://learn.valve.city/rpc";function o(t){return t.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;").replace(/'/g,"&#39;")}function U(t,n){const a=t&&n?` <span class="footer-sep">·</span> <a href="${o(n)}" target="_blank" rel="noopener noreferrer">${o(t)}</a>`:"";return`
    <footer class="footer">
      <a href="${o(De)}" target="_blank" rel="noopener noreferrer">Learn how this works → learn.valve.city/rpc</a>${a}
    </footer>
  `}function Ae(t){t.innerHTML=`
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
  `;const n=t.querySelector("#content"),a=Array.from(t.querySelectorAll("[data-nav]"));return{contentEl:n,setActiveNav:l=>{for(const p of a)p.classList.toggle("active",p.dataset.nav===l)}}}function J(t){return Number.isFinite(t)?t.toLocaleString("en-US"):"—"}function Ne(t){return Number.isFinite(t)?`${t.toFixed(1)}%`:"—"}function Me(t){if(!Number.isFinite(t)||t<0)return"—";if(t<60)return`~${Math.round(t)}s`;const n=Math.round(t/60),a=Math.floor(n/60),e=n%60;if(a===0)return`~${e}m`;if(a<48)return`~${a}h ${e}m`;const l=Math.floor(a/24),p=a%24;return`~${l}d ${p}h`}function R(t,n){return`<span class="badge badge-${n}">${o(t)}</span>`}function le(t){return`<span class="dot dot-${t}"></span>`}const de=["B","KB","MB","GB","TB","PB"];function K(t){if(!Number.isFinite(t)||t<0)return"—";if(t===0)return"0 B";let n=t,a=0;for(;n>=1024&&a<de.length-1;)n/=1024,a++;const e=n<10?2:n<100?1:0;return`${n.toFixed(e)} ${de[a]}`}async function me(t){try{return await navigator.clipboard.writeText(t),!0}catch{return!1}}function Z(t,n){t.addEventListener("click",a=>{const e=a.target.closest("[data-action]");if(!e||!t.contains(e))return;const l=e.dataset.action;l&&n(l,e,a)})}const Ue=85,ae={exec:"Execution",beacon:"Beacon"};function qe(t,n){let a=!1,e=null,l=null,p=null,T=null,g=null,H=null,E=null,B=null;const d={exec:null,beacon:null};let $=null;t.innerHTML=`<h1>Dashboard: ${o(n)}</h1><div id="dash-body"><p class="muted">Loading…</p></div><div id="dash-footer">${U()}</div>`;const P=t.querySelector("#dash-body"),f=t.querySelector("#dash-footer");P.addEventListener("click",r=>{const u=r.target.closest("[data-action]");if(!u||!P.contains(u))return;const v=u.dataset.action;if(v==="svc-action"){const b=u.dataset.svc,C=u.dataset.kind;b&&C&&N(b,C)}else if(v==="open-clear"){const b=u.dataset.svc;b&&V(b)}else if(v==="copy"){const b=u.dataset.copy;b&&F(u,b)}else v==="retry-du"?h():v==="retry-endpoints"&&c()}),s();async function s(){let r,u;try{const[b,C]=await Promise.all([Y(),X()]);r=b.find(D=>D.id===n),u=C}catch(b){if(a)return;P.innerHTML=`<p class="error">Failed to load target: ${o(String(b))}</p>`;return}if(a)return;if(!r){P.innerHTML=`<p class="error">Target "${o(n)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!r.wire){P.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(n)}">Run the setup wizard →</a></p>`;return}const v=u==null?void 0:u.networks.find(b=>b.ChainID===r.wire.ChainID);v&&(f.innerHTML=U(v.Name,v.LearnURL)),P.innerHTML='<p class="muted">Connecting…</p>',e=xe(n,b=>{a||(I(b),l=b,p=b,L())}),h(),c()}async function h(){H=null;try{g=await Ie(n)}catch(r){g=null,H=String(r instanceof Error?r.message:r)}a||L()}async function c(){B=null;try{E=await Le(n)}catch(r){E=null,B=String(r instanceof Error?r.message:r)}a||L()}function I(r){if(!l)return;const u=(new Date(r.at).getTime()-new Date(l.at).getTime())/1e3,v=r.execHead-l.execHead;if(u>0&&v>=0){const b=v/u;T=T===null?b:T*.7+b*.3}}function L(){if(!p)return;const r=p;P.innerHTML=`
      <div class="card-grid">
        ${q(r)}
        ${W(r)}
        ${O(r)}
        ${i(r)}
        ${m(r)}
        ${y()}
        ${k(r)}
      </div>
      <p class="muted small">Last updated ${o(new Date(r.at).toLocaleTimeString())}</p>
    `}function M(r){const v=r.refHead>0?r.refHead-r.execHead:null,b=v!==null&&v>0&&T&&T>0?Me(v/T):v!==null&&v<=0?"caught up":"—";return{lag:v,eta:b}}function q(r){const{lag:u,eta:v}=M(r);return`
      <div class="card">
        <h3>Execution sync</h3>
        <p>${r.execSyncing?R("syncing","warn"):R("synced","ok")}</p>
        <dl class="stat-list">
          <div><dt>Local head</dt><dd>${J(r.execHead)}</dd></div>
          <div><dt>Reference head</dt><dd>${u!==null?J(r.refHead):"unavailable"}</dd></div>
          <div><dt>Lag</dt><dd>${u!==null?J(Math.max(u,0))+" blocks":"—"}</dd></div>
          <div><dt>ETA</dt><dd>${v}</dd></div>
        </dl>
      </div>
    `}function W(r){return`
      <div class="card">
        <h3>Beacon sync</h3>
        <p>${r.beaconDistance===0?R("synced","ok"):R("syncing","warn")}</p>
        <dl class="stat-list">
          <div><dt>Slot</dt><dd>${J(r.beaconSlot)}</dd></div>
          <div><dt>Distance</dt><dd>${J(r.beaconDistance)}</dd></div>
        </dl>
      </div>
    `}function O(r){return`
      <div class="card">
        <h3>Peers</h3>
        <dl class="stat-list">
          <div><dt>Execution</dt><dd>${J(r.execPeers)}</dd></div>
          <div><dt>Beacon</dt><dd>${J(r.beaconPeers)}</dd></div>
        </dl>
      </div>
    `}function i(r){const u=r.diskUsedPct>=Ue;return`
      <div class="card ${u?"card-warn":""}">
        <h3>Disk</h3>
        <div class="meter"><div class="meter-fill ${u?"meter-warn":""}" style="width:${Math.min(r.diskUsedPct,100)}%"></div></div>
        <p>${Ne(r.diskUsedPct)} used</p>
      </div>
    `}function m(r){if(H)return`
        <div class="card card-warn">
          <h3>Storage</h3>
          <p class="error small">${o(H)}</p>
          <button class="btn btn-ghost" data-action="retry-du">Retry</button>
        </div>
      `;if(!g)return'<div class="card"><h3>Storage</h3><p class="muted">Loading…</p></div>';const u=g.ExpectedExecBytes>0?Math.min(g.ExecBytes/g.ExpectedExecBytes*100,100):0,v=g.ExpectedBeaconBytes>0?Math.min(g.BeaconBytes/g.ExpectedBeaconBytes*100,100):0,{lag:b,eta:C}=M(r),D=b!==null&&b>0&&T!==null&&T>0;return`
      <div class="card">
        <h3>Storage</h3>
        <p class="muted small">Estimate — varies by client and pruning.</p>
        <p class="muted small">Execution — ${K(g.ExecBytes)} of ~${K(g.ExpectedExecBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${u}%"></div></div>
        ${D?`<p class="muted small">Estimated time remaining: ${o(C)}</p>`:""}
        <p class="muted small">Beacon — ${K(g.BeaconBytes)} of ~${K(g.ExpectedBeaconBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${v}%"></div></div>
        <dl class="stat-list">
          <div><dt>Disk free</dt><dd>${K(g.DiskFreeBytes)}</dd></div>
          <div><dt>Sync (snapshot)</dt><dd>${o(g.SyncLabel)}</dd></div>
          <div><dt>Sync (genesis)</dt><dd>${o(g.GenesisSyncLabel)}</dd></div>
        </dl>
      </div>
    `}function y(){if(B)return`
        <div class="card card-warn">
          <h3>Endpoints</h3>
          <p class="error small">${o(B)}</p>
          <button class="btn btn-ghost" data-action="retry-endpoints">Retry</button>
        </div>
      `;if(!E)return'<div class="card"><h3>Endpoints</h3><p class="muted">Loading…</p></div>';const r=E,u=r.ExecReachable&&!r.ChainIDMatches?`<p class="error small">Exec responded, but its chain id doesn't match this target's wire config.</p>`:"",v=r.Access==="ssh"?`
          <p class="muted small">These URLs are local to the server; use the tunnel or your own reverse proxy to reach them from elsewhere.</p>
          <div class="endpoint-row">
            <code class="endpoint-url">${o(r.TunnelHint)}</code>
            <button class="btn btn-ghost" data-action="copy" data-copy="${o(r.TunnelHint)}">Copy</button>
          </div>
        `:"";return`
      <div class="card">
        <h3>Endpoints</h3>
        <div class="endpoint-row">
          ${le(r.ExecReachable?"ok":"bad")}
          <code class="endpoint-url">${o(r.ExecHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${o(r.ExecHTTP)}">Copy</button>
        </div>
        <div class="endpoint-row">
          ${le(r.BeaconReachable?"ok":"bad")}
          <code class="endpoint-url">${o(r.BeaconHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${o(r.BeaconHTTP)}">Copy</button>
        </div>
        ${u}
        ${v}
      </div>
    `}function x(r,u){const v=ae[r],b=d[r],C=(D,be,ge)=>`<button class="btn btn-ghost" data-action="svc-action" data-svc="${r}" data-kind="${D}" ${b!==null||ge?"disabled":""}>${b===D?S():o(be)}</button>`;return`
      <div class="service-row">
        <span>${o(v)} ${u?R("active","ok"):R("down","bad")}</span>
        <div class="service-actions">
          ${C("start","Start",u)}
          ${C("stop","Stop",!u)}
          ${C("restart","Restart",!1)}
          <button class="btn btn-danger" data-action="open-clear" data-svc="${r}" ${b!==null?"disabled":""}>Clear…</button>
        </div>
      </div>
    `}function k(r){return`
      <div class="card">
        <h3>Services</h3>
        ${x("exec",r.execActive)}
        ${x("beacon",r.beaconActive)}
        ${$?`<p class="error small">${o($)}</p>`:""}
        <p class="card-links">
          <a href="#/logs/${encodeURIComponent(n)}">View logs →</a>
          <a href="#/security/${encodeURIComponent(n)}">Security →</a>
        </p>
      </div>
    `}function S(){return'<span class="spinner" aria-label="working"></span>'}async function N(r,u){if(d[r]===null){d[r]=u,$=null,L();try{await ke(n,r,u)}catch(v){$=`${ae[r]} ${u} failed: ${v instanceof Error?v.message:String(v)}`}d[r]=null,a||L()}}async function F(r,u){const v=await me(u),b=r.textContent;r.textContent=v?"Copied!":"Copy failed",setTimeout(()=>{a||(r.textContent=b)},1500)}function V(r){const u=ae[r],v=g?K(r==="exec"?g.ExecBytes:g.BeaconBytes):"unknown (disk usage hasn't loaded)";_(`
        <h2>Clear ${o(u)} data</h2>
        <p class="error">
          This stops the ${o(u.toLowerCase())} service, deletes its chain data under the
          node's data directory (current size: ${o(v)}), and starts it again. A full
          resync is required afterward.
        </p>
        <p>Type <code>${o(r)}</code> to confirm.</p>
        <input type="text" id="clear-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="clear-confirm-btn" disabled>Clear and resync</button>
        </div>
      `,D=>{if(D==="cancel"){z();return}D==="confirm"&&w(r)});const b=document.getElementById("clear-confirm-input"),C=document.getElementById("clear-confirm-btn");b==null||b.addEventListener("input",()=>{C&&(C.disabled=b.value.trim()!==r)}),b==null||b.focus()}async function w(r){const u=document.getElementById("clear-confirm-btn");u&&(u.disabled=!0,u.textContent="Clearing…");try{await Ce(n,r),z(),h()}catch(v){const b=document.querySelector("#clear-modal .modal");if(b){const C=document.createElement("p");C.className="error small",C.textContent=`Clear failed: ${v instanceof Error?v.message:String(v)}`,b.appendChild(C)}u&&(u.disabled=!1,u.textContent="Clear and resync")}}function _(r,u){z();const v=document.createElement("div");v.className="modal-overlay",v.id="clear-modal",v.innerHTML=`<div class="modal">${r}</div>`,v.addEventListener("click",b=>{const C=b.target.closest("[data-modal-action]");C!=null&&C.dataset.modalAction&&u(C.dataset.modalAction),b.target===v&&u("cancel")}),document.body.appendChild(v)}function z(){var r;(r=document.getElementById("clear-modal"))==null||r.remove()}return()=>{a=!0,e==null||e(),z()}}const ue=500,pe="valve-node.explain-consent";function Oe(t,n){let a=!1,e=null;const l=[];t.innerHTML=`
    <h1>Logs: ${o(n)}</h1>
    <div id="logs-body"><p class="muted">Loading…</p></div>
    <div id="logs-footer">${U()}</div>
  `;const p=t.querySelector("#logs-body"),T=t.querySelector("#logs-footer");Z(t,s=>{s==="explain"&&B()}),g();async function g(){let s,h;try{const[I,L]=await Promise.all([Y(),X()]);s=I.find(M=>M.id===n),h=L}catch(I){if(a)return;p.innerHTML=`<p class="error">Failed to load target: ${o(String(I))}</p>`;return}if(a)return;if(!s){p.innerHTML=`<p class="error">Target "${o(n)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!s.wire){p.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(n)}">Run the setup wizard →</a></p>`;return}const c=h==null?void 0:h.networks.find(I=>I.ChainID===s.wire.ChainID);c&&(T.innerHTML=U(c.Name,c.LearnURL));try{const I=await Se(n,200);if(a)return;l.push(...I)}catch(I){if(a)return;p.innerHTML=`<p class="error">Failed to load logs: ${o(String(I))}</p>`;return}H(),e=Ee(n,I=>{a||(l.push(I),l.length>ue&&l.splice(0,l.length-ue),H())})}function H(){const s=l.filter(c=>c.severity==="error"||c.severity==="critical");p.innerHTML=`
      <div class="logs-layout">
        <section class="logs-tail">
          <div class="logs-tail-head">
            <h2>Live tail</h2>
            <button class="btn" data-action="explain">Explain with AI</button>
          </div>
          <div class="log-lines">${l.map(E).join("")}</div>
        </section>
        <section class="logs-errors">
          <h2>Error feed ${R(String(s.length),s.length?"bad":"neutral")}</h2>
          <div class="log-lines">${s.length?s.slice().reverse().map(E).join(""):'<p class="muted">No errors seen yet.</p>'}</div>
        </section>
      </div>
    `;const h=p.querySelector(".log-lines");h&&(h.scrollTop=h.scrollHeight)}function E(s){const h=s.severity||"info",c=s.learnUrl?` <a href="${o(s.learnUrl)}" target="_blank" rel="noopener noreferrer">learn →</a>`:"";return`
      <div class="log-line log-${o(h)}">
        <span class="log-time">${o(new Date(s.at).toLocaleTimeString())}</span>
        <span class="log-unit">${o(s.unit)}</span>
        <span class="log-sev">${o(h)}</span>
        <span class="log-text">${o(s.line)}</span>
        ${s.explain?`<div class="log-explain">${o(s.explain)}${c}</div>`:""}
      </div>
    `}async function B(){const s=l.filter(c=>c.severity==="error"||c.severity==="critical").map(c=>c.line).slice(-40);if(!(localStorage.getItem(pe)==="1")){d(s);return}await $(s)}function d(s){const h=s.length?`<pre class="explain-excerpt">${s.map(c=>o(c)).join(`
`)}</pre>`:'<p class="muted">No recent error lines are loaded yet — the server will auto-select its own recent error/critical lines instead.</p>';P(`
      <h2>Send logs to your AI provider?</h2>
      <p>
        The excerpt below will be sent to the AI provider configured in
        <a href="#/settings">Settings</a> to generate a plain-English
        explanation. This happens every time you click "Explain with AI";
        this confirmation only shows once per browser.
      </p>
      ${h}
      <div class="modal-actions">
        <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        <button class="btn btn-primary" data-modal-action="proceed">Send to AI provider</button>
      </div>
    `,c=>{c==="proceed"?(localStorage.setItem(pe,"1"),f(),$(s)):f()})}async function $(s){P('<h2>Explain with AI</h2><p class="muted">Asking the AI provider…</p>',()=>{});try{const h=s.length?await ce(n,s):await ce(n);if(a)return;P(`
        <h2>Explanation</h2>
        <div class="explain-text">${o(h.text)}</div>
        <details class="advanced">
          <summary>What was sent</summary>
          <pre class="explain-excerpt">${h.sentExcerpt.map(c=>o(c)).join(`
`)||"(no log lines — general question only)"}</pre>
        </details>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,c=>{c==="close"&&f()})}catch(h){if(a)return;if(h instanceof oe&&h.status===409){P(`
          <h2>No AI provider configured</h2>
          <p>Set a provider and key in <a href="#/settings">Settings</a>, then try again.</p>
          <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
        `,c=>{c==="close"&&f()});return}P(`
        <h2>Explain failed</h2>
        <p class="error">${o(h instanceof Error?h.message:String(h))}</p>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,c=>{c==="close"&&f()})}}function P(s,h){f();const c=document.createElement("div");c.className="modal-overlay",c.id="explain-modal",c.innerHTML=`<div class="modal">${s}</div>`,c.addEventListener("click",I=>{const L=I.target.closest("[data-modal-action]");L!=null&&L.dataset.modalAction&&h(L.dataset.modalAction),I.target===c&&h("cancel")}),document.body.appendChild(c)}function f(){var s;(s=document.getElementById("explain-modal"))==null||s.remove()}return()=>{a=!0,e==null||e(),f()}}function je(t,n){let a=!1,e=[],l=null,p=!1,T=!1;t.innerHTML=`<h1>Security: ${o(n)}</h1><div id="sec-body"><p class="muted">Loading…</p></div><div id="sec-footer">${U()}</div>`;const g=t.querySelector("#sec-body"),H=t.querySelector("#sec-footer");Z(t,(f,s)=>{var h;if(f==="rerun")B();else if(f==="toggle")(h=s.closest(".check-item"))==null||h.classList.toggle("expanded");else if(f==="copy"){const c=s.dataset.copy;c&&P(s,c)}}),E();async function E(){let f,s;try{const[c,I]=await Promise.all([Y(),X()]);f=c.find(L=>L.id===n),s=I}catch(c){if(a)return;g.innerHTML=`<p class="error">Failed to load target: ${o(String(c))}</p>`;return}if(a)return;if(!f){g.innerHTML=`<p class="error">Target "${o(n)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!f.wire){g.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(n)}">Run the setup wizard →</a></p>`;return}const h=s==null?void 0:s.networks.find(c=>c.ChainID===f.wire.ChainID);h&&(H.innerHTML=U(h.Name,h.LearnURL)),await B()}async function B(){p=!0,l=null,d();try{e=await He(n),T=!0}catch(f){l=String(f instanceof Error?f.message:f)}p=!1,a||d()}function d(){g.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(n)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Every check here is a live, read-only probe run on the target — nothing is ever changed
          automatically. Each "Fix" is a copy-paste command for you to review and run yourself.
        </p>
        <button class="btn" data-action="rerun" ${p?"disabled":""}>${p?"Re-running…":"Re-run checks"}</button>
      </div>
      ${l?`<p class="error">${o(l)}</p>`:""}
      ${!T&&p?'<p class="muted">Loading…</p>':e.length?`<ul class="check-list">${e.map($).join("")}</ul>`:T?'<p class="muted">No checks returned.</p>':""}
    `}function $(f){const s=f.Status==="pass"?"ok":f.Status==="fail"?"bad":f.Status==="warn"?"warn":"neutral";return`
      <li class="check-item">
        <button class="check-head" data-action="toggle" type="button">
          ${R(f.Status,s)}
          <strong>${o(f.Title)}</strong>
          <span class="muted small check-detail-inline">${o(f.Detail)}</span>
        </button>
        <div class="check-body">
          <details>
            <summary>Why this matters</summary>
            <p class="muted small">${o(f.Why)}</p>
          </details>
          ${f.Fix?`
                <details open>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${o(f.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${o(f.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function P(f,s){const h=await me(s),c=f.textContent;f.textContent=h?"Copied!":"Copy failed",setTimeout(()=>{a||(f.textContent=c)},1500)}return()=>{a=!0}}const Fe=[{value:"",label:"None"},{value:"gemini",label:"Gemini"},{value:"groq",label:"Groq"},{value:"ollama",label:"Ollama"}];function _e(t){let n=!1,a=!1,e=!1,l=null,p=!1,T=null;t.innerHTML=`<h1>Settings</h1><div id="settings-body"><p class="muted">Loading…</p></div>${U()}`;const g=t.querySelector("#settings-body");Z(t,d=>{if(d==="save"&&B(),d==="clear-key"){if(!T)return;a=!0;const $=t.querySelector("#ai-key");$&&($.value=""),E(T)}}),H();async function H(){try{const d=await Re();if(n)return;T=d,E(d)}catch(d){if(n)return;g.innerHTML=`<p class="error">Failed to load settings: ${o(String(d))}</p>`}}function E(d){var f,s;const $=Fe.map(h=>`<option value="${h.value}" ${d.aiProvider===h.value?"selected":""}>${o(h.label)}</option>`).join("");g.innerHTML=`
      <form class="card" id="settings-form" onsubmit="return false">
        <label>
          AI provider
          <select id="ai-provider">${$}</select>
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
        ${l?`<p class="error">${o(l)}</p>`:""}
        ${p?'<p class="ok">Saved.</p>':""}
        <button class="btn btn-primary" type="button" data-action="save" ${e?"disabled":""}>${e?"Saving…":"Save"}</button>
      </form>
    `;const P=t.querySelector("#ai-key");P==null||P.addEventListener("input",()=>{a=!0,p=!1}),(f=t.querySelector("#ai-provider"))==null||f.addEventListener("change",()=>{p=!1}),(s=t.querySelector("#ref-rpc-base"))==null||s.addEventListener("input",()=>{p=!1})}async function B(){const d=t.querySelector("#ai-provider"),$=t.querySelector("#ai-key"),P=t.querySelector("#ref-rpc-base");if(!d||!$||!P||!T)return;const f={aiProvider:d.value,refRpcBase:P.value.trim()};a&&(f.aiKey=$.value),e=!0,l=null,p=!1,E(T);try{const s=await Be(f);if(n)return;T=s,a=!1,e=!1,p=!0,E(s)}catch(s){if(n)return;e=!1,l=String(s instanceof Error?s.message:s),E(T)}}return()=>{n=!0}}const ze="local";function Je(t){let n=!1;t.innerHTML=`
    <h1>Targets</h1>
    <div id="targets-body"><p class="muted">Loading…</p></div>
    ${U()}
  `;const a=t.querySelector("#targets-body");Z(t,(d,$)=>{p(d,$)}),e();async function e(){try{const[d,$]=await Promise.all([Y(),X()]);if(n)return;l(d,$)}catch(d){if(n)return;a.innerHTML=`<p class="error">Failed to load targets: ${o(String(d))}</p>`}}function l(d,$){const P=d.find(c=>c.mode==="local"),f=d.filter(c=>c.mode==="ssh"),s=P?fe(P,$):`
        <div class="card">
          <h2>This machine</h2>
          ${Ge()}
          <button class="btn" data-action="add-local">Add this machine as a target</button>
        </div>
      `,h=f.length?f.map(c=>fe(c,$)).join(""):'<p class="muted">No SSH targets yet.</p>';a.innerHTML=`
      <section class="section">${s}</section>
      <section class="section">
        <h2>Servers over SSH</h2>
        <div class="card-grid">${h}</div>
        ${We()}
      </section>
    `}async function p(d,$){if(d==="add-local"){await T();return}if(d==="delete-target"){const P=$.dataset.id;if(!P||!confirm(`Remove target "${P}"? This does not touch anything already running on it.`))return;await g(P);return}d==="add-ssh"&&await H()}async function T(){B();try{await ie({id:ze,mode:"local"}),await e()}catch(d){E(d)}}async function g(d){try{await we(d),await e()}catch($){E($)}}async function H(){const d=t.querySelector("#ssh-host"),$=t.querySelector("#ssh-user"),P=t.querySelector("#ssh-key"),f=t.querySelector("#ssh-port"),s=t.querySelector("#ssh-id");if(!d||!$||!P||!f||!s)return;const h=d.value.trim(),c=$.value.trim(),I=P.value.trim(),L=f.value.trim(),M=s.value.trim();if(B(),!h||!c||!I){E(new Error("host, user, and key path are required"));return}const q=M||Ve(h),W={Host:h,User:c,KeyPath:I};if(L){const i=Number.parseInt(L,10);if(!Number.isFinite(i)||i<=0){E(new Error("port must be a positive number"));return}W.Port=i}const O=t.querySelector("#ssh-submit");O&&(O.disabled=!0,O.textContent="Connecting…");try{await ie({id:q,mode:"ssh",ssh:W}),await e()}catch(i){E(i),O&&(O.disabled=!1,O.textContent="Add server")}}function E(d){let $=t.querySelector("#targets-error");$||(a.insertAdjacentHTML("afterbegin",'<p id="targets-error" class="error"></p>'),$=t.querySelector("#targets-error")),$.textContent=String(d instanceof Error?d.message:d)}function B(){var d;(d=t.querySelector("#targets-error"))==null||d.remove()}return()=>{n=!0}}function fe(t,n){const a=t.wire,e=t.mode==="local"?"this machine":"SSH",l=t.mode==="ssh"&&t.ssh?`${o(t.ssh.User)}@${o(t.ssh.Host)}`:e;let p,T;if(!a)p=R("not set up","neutral"),T=`<a class="btn" href="#/setup/${encodeURIComponent(t.id)}">Run setup wizard</a>`;else{const g=n.networks.find(E=>E.ChainID===a.ChainID),H=g?g.Name:`chain ${a.ChainID}`;p=`${R(H,"ok")} ${R(a.ExecID,"neutral")} ${R(a.BeaconID,"neutral")}${a.Archive?" "+R("archive","warn"):""}`,T=`
      <a class="btn" href="#/dash/${encodeURIComponent(t.id)}">Dashboard</a>
      <a class="btn" href="#/logs/${encodeURIComponent(t.id)}">Logs</a>
      <a class="btn btn-ghost" href="#/setup/${encodeURIComponent(t.id)}">Re-run setup</a>
    `}return`
    <div class="card">
      <h2>${o(t.id)}</h2>
      <p class="muted">${l}</p>
      <p>${p}</p>
      <div class="card-actions">
        ${T}
        <button class="btn btn-danger" data-action="delete-target" data-id="${o(t.id)}">Remove</button>
      </div>
    </div>
  `}function We(){return`
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
  `}function Ke(){const t=navigator.userAgentData,n=(t==null?void 0:t.platform)||navigator.platform||navigator.userAgent;return/mac|win/i.test(n)&&!/linux|android/i.test(n)}function Ge(){return Ke()?`
      <p class="banner banner-warn">
        macOS and Windows are not supported node hosts — use this machine as a controller and add a
        Linux server over SSH.
      </p>
    `:'<p class="muted">The machine running valve-node. Setup only works on a Linux target.</p>'}function Ve(t){return t.toLowerCase().replace(/[^a-z0-9]+/g,"-").replace(/^-+|-+$/g,"")||"target"}const re=[{id:"preflight",title:"Preflight checks"},{id:"toolchain",title:"Ensure git + build toolchains"},{id:"install-exec",title:"Install execution client"},{id:"install-beacon",title:"Install beacon client"},{id:"wire",title:"Write JWT secret and systemd units"},{id:"start",title:"Start execution and beacon services"},{id:"handshake",title:"Verify execution/beacon handshake"}],ee=8545,te=5052,ne=30303,Xe=[369,943,1],he={369:"default",943:"practise here first"};function Ye(t,n){let a=!1;const e={targetId:n,step:"network",catalog:null,loadError:null,chainId:369,execId:null,beaconId:null,archive:!1,dataDir:"",jwtPath:"",execHTTPPort:"",beaconHTTPPort:"",execP2PPort:"",execHTTPPortError:null,beaconHTTPPortError:null,execP2PPortError:null,starting:!1,startError:null,events:[],streamStop:null};t.innerHTML=`<h1>Setup: ${o(n)}</h1><div id="wizard-body"><p class="muted">Loading catalog…</p></div><div id="wizard-footer">${U()}</div>`;const l=t.querySelector("#wizard-body"),p=t.querySelector("#wizard-footer");Z(t,(i,m)=>{h(i,m)}),T();async function T(){try{const[i,m]=await Promise.all([X(),Y()]);if(a)return;e.catalog=i;const y=m.find(x=>x.id===n);y!=null&&y.wire&&(e.chainId=y.wire.ChainID,e.execId=y.wire.ExecID,e.beaconId=y.wire.BeaconID,e.archive=y.wire.Archive,y.wire.ExecHTTPPort&&(e.execHTTPPort=String(y.wire.ExecHTTPPort)),y.wire.BeaconHTTPPort&&(e.beaconHTTPPort=String(y.wire.BeaconHTTPPort)),y.wire.ExecP2PPort&&(e.execP2PPort=String(y.wire.ExecP2PPort))),g()}catch(i){if(a)return;e.loadError=String(i instanceof Error?i.message:i),g()}}function g(){if(e.loadError){l.innerHTML=`<p class="error">Failed to load: ${o(e.loadError)}</p>`;return}e.catalog&&(l.innerHTML=`
      ${O(e.step)}
      ${E()}
    `,H())}function H(){var m;const i=(m=e.catalog)==null?void 0:m.networks.find(y=>y.ChainID===e.chainId);p.innerHTML=i?U(i.Name,i.LearnURL):U()}function E(){switch(e.step){case"network":return B();case"clients":return d();case"mode":return P();case"review":return f();case"run":return s()}}function B(){const i=e.catalog;return`
      <section>
        <h2>1. Choose a network</h2>
        <div class="card-grid">${Xe.map(y=>{const x=i.networks.find(N=>N.ChainID===y);if(!x)return"";const k=e.chainId===y,S=he[y]?R(he[y],y===369?"ok":"warn"):"";return`
        <button class="card card-selectable ${k?"selected":""}" data-action="pick-network" data-chain-id="${y}" type="button">
          <h3>${o(x.Name)} <span class="muted">(chain ${y})</span></h3>
          ${S}
          <p class="muted small">Checkpoint sync from ${o(x.CheckpointURL)}</p>
        </button>
      `}).join("")}</div>
        <div class="wizard-actions">
          <button class="btn" data-action="goto-clients" ${e.chainId===null?"disabled":""}>Next: clients</button>
        </div>
      </section>
    `}function d(){const i=e.catalog,m=i.networks.find(k=>k.ChainID===e.chainId);if(!m)return'<p class="error">Unknown network.</p>';(e.execId===null||!m.ExecClients.includes(e.execId))&&(e.execId=m.ExecClients[0]??null),(e.beaconId===null||!m.BeaconClients.includes(e.beaconId))&&(e.beaconId=m.BeaconClients[0]??null);const y=m.ExecClients.map(k=>$(k,i,e.execId)).join(""),x=m.BeaconClients.map(k=>$(k,i,e.beaconId)).join("");return`
      <section>
        <h2>2. Choose your client pair</h2>
        <p class="muted">Only combinations known to work on ${o(m.Name)} are offered.</p>
        <label>
          Execution client
          <select id="exec-select" data-action="pick-exec">${y}</select>
        </label>
        <label>
          Beacon client
          <select id="beacon-select" data-action="pick-beacon">${x}</select>
        </label>
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-network">Back</button>
          <button class="btn" data-action="goto-mode">Next: mode</button>
        </div>
      </section>
    `}function $(i,m,y){const x=m.clients.find(S=>S.id===i),k=x?`${x.id} (${x.toolchain})`:i;return`<option value="${o(i)}" ${i===y?"selected":""}>${o(k)}</option>`}function P(){const i=e.chainId!==null?`/var/lib/valve-node/${e.chainId}`:"";return`
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
            Data directory <span class="muted">(default: ${o(i)})</span>
            <input id="data-dir-input" type="text" placeholder="${o(i)}" value="${o(e.dataDir)}" />
          </label>
          <label>
            JWT secret path <span class="muted">(default: &lt;data dir&gt;/jwt.hex)</span>
            <input id="jwt-path-input" type="text" placeholder="${o(i)}/jwt.hex" value="${o(e.jwtPath)}" />
          </label>
          <label>
            Execution HTTP port <span class="muted">(default: ${ee})</span>
            <input id="exec-http-port-input" type="text" inputmode="numeric" placeholder="${ee}" value="${o(e.execHTTPPort)}" />
          </label>
          ${e.execHTTPPortError?`<p class="error small">${o(e.execHTTPPortError)}</p>`:""}
          <label>
            Beacon HTTP port <span class="muted">(default: ${te})</span>
            <input id="beacon-http-port-input" type="text" inputmode="numeric" placeholder="${te}" value="${o(e.beaconHTTPPort)}" />
          </label>
          ${e.beaconHTTPPortError?`<p class="error small">${o(e.beaconHTTPPortError)}</p>`:""}
          <label>
            Execution p2p port <span class="muted">(default: ${ne})</span>
            <input id="exec-p2p-port-input" type="text" inputmode="numeric" placeholder="${ne}" value="${o(e.execP2PPort)}" />
          </label>
          ${e.execP2PPortError?`<p class="error small">${o(e.execP2PPortError)}</p>`:""}
          <p class="muted small">
            Leave any of these blank to use the default. The engine API port (8551) is fixed and
            loopback-only — it isn't configurable.
          </p>
        </details>
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-clients">Back</button>
          <button class="btn" data-action="goto-review">Next: review</button>
        </div>
      </section>
    `}function f(){const m=e.catalog.networks.find(w=>w.ChainID===e.chainId),y=e.dataDir||`/var/lib/valve-node/${e.chainId}`,x=e.jwtPath||`${y}/jwt.hex`,k=re.map(w=>`<li>${o(w.title)}</li>`).join(""),S=q(e.execHTTPPort,ee),N=q(e.beaconHTTPPort,te),F=q(e.execP2PPort,ne),V=S||N||F?`<tr><th>Non-default ports</th><td>${[S?`exec HTTP ${S}`:null,N?`beacon HTTP ${N}`:null,F?`exec p2p ${F}`:null].filter(w=>w!==null).map(o).join(", ")}</td></tr>`:"";return`
      <section>
        <h2>4. Review</h2>
        <table class="review-table">
          <tbody>
            <tr><th>Target</th><td>${o(e.targetId)}</td></tr>
            <tr><th>Network</th><td>${o((m==null?void 0:m.Name)??String(e.chainId))} (chain ${e.chainId})</td></tr>
            <tr><th>Execution client</th><td>${o(e.execId??"")}</td></tr>
            <tr><th>Beacon client</th><td>${o(e.beaconId??"")}</td></tr>
            <tr><th>Mode</th><td>${e.archive?"Archive":"Full"}</td></tr>
            <tr><th>Data directory</th><td><code>${o(y)}</code></td></tr>
            <tr><th>JWT secret path</th><td><code>${o(x)}</code></td></tr>
            ${V}
          </tbody>
        </table>
        <p class="muted small">
          There is no preview API for the exact files/units that will be
          written — the list below is the fixed step sequence setup always
          runs; the actual commands and file contents stream live once you
          start.
        </p>
        <ol class="step-preview">${k}</ol>
        ${e.startError?`<p class="error">${o(e.startError)}</p>`:""}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-mode">Back</button>
          <button class="btn btn-primary" data-action="start-setup" ${e.starting?"disabled":""}>
            ${e.starting?"Starting…":"Start setup"}
          </button>
        </div>
      </section>
    `}function s(){const m=e.catalog.networks.find(w=>w.ChainID===e.chainId),y=m==null?void 0:m.LearnURL,x=new Set(e.events.filter(w=>w.done).map(w=>w.stepId)),k=new Set(e.events.filter(w=>w.err).map(w=>w.stepId)),S=new Map;for(const w of e.events){if(!w.line)continue;const _=S.get(w.stepId)??[];_.push(w.line),S.set(w.stepId,_)}const N=re.map(w=>{var C;const _=x.has(w.id),z=k.has(w.id),r=z?R("failed","bad"):_?R("done","ok"):R("pending","neutral"),u=(S.get(w.id)??[]).slice(-5),v=(C=e.events.find(D=>D.stepId===w.id&&D.err))==null?void 0:C.err,b=w.id==="handshake"?`<p class="muted small">"Talking" means the beacon client can reach the execution client's Engine API over the shared JWT secret and both report the same head — the sign your node is wired correctly.${y?` <a href="${o(y)}" target="_blank" rel="noopener noreferrer">Learn more →</a>`:""}</p>`:"";return`
        <li class="step-row ${_?"step-done":""} ${z?"step-error":""}">
          <div class="step-head">${r} <strong>${o(w.title)}</strong></div>
          ${b}
          ${u.length?`<pre class="step-log">${u.map(D=>o(D)).join(`
`)}</pre>`:""}
          ${v?`<p class="error small">${o(v)}</p>`:""}
        </li>
      `}).join(""),F=e.events.some(w=>w.err),V=re.every(w=>x.has(w.id))||e.events.some(w=>w.stepId==="handshake"&&w.done);return`
      <section>
        <h2>5. Running setup</h2>
        <ol class="step-list">${N}</ol>
        ${V&&!F?`<p class="ok">Setup complete. <a href="#/dash/${encodeURIComponent(e.targetId)}">Open the dashboard →</a></p>`:""}
        ${F?'<button class="btn" data-action="start-setup">Retry setup</button>':""}
      </section>
    `}function h(i,m){switch(i){case"pick-network":e.chainId=Number(m.dataset.chainId),e.execId=null,e.beaconId=null,g();break;case"goto-network":e.step="network",g();break;case"goto-clients":if(e.chainId===null)return;e.step="clients",g();break;case"goto-mode":c(),e.step="mode",g();break;case"goto-review":if(I(),e.execHTTPPortError||e.beaconHTTPPortError||e.execP2PPortError){g();break}e.step="review",g();break;case"start-setup":W();break}}function c(){const i=t.querySelector("#exec-select"),m=t.querySelector("#beacon-select");i&&(e.execId=i.value),m&&(e.beaconId=m.value)}function I(){const i=t.querySelectorAll('input[name="mode"]');for(const N of Array.from(i))N.checked&&(e.archive=N.value==="archive");const m=t.querySelector("#data-dir-input"),y=t.querySelector("#jwt-path-input");m&&(e.dataDir=m.value.trim()),y&&(e.jwtPath=y.value.trim());const x=t.querySelector("#exec-http-port-input"),k=t.querySelector("#beacon-http-port-input"),S=t.querySelector("#exec-p2p-port-input");x&&(e.execHTTPPort=x.value.trim()),k&&(e.beaconHTTPPort=k.value.trim()),S&&(e.execP2PPort=S.value.trim()),e.execHTTPPortError=M(e.execHTTPPort).error??null,e.beaconHTTPPortError=M(e.beaconHTTPPort).error??null,e.execP2PPortError=M(e.execP2PPort).error??null}const L=/^\d+$/;function M(i){if(!i)return{};if(!L.test(i))return{error:"Enter a whole number (no decimals, signs, or other characters)."};const m=Number(i);return!Number.isInteger(m)||m<1||m>65535?{error:"Port must be between 1 and 65535."}:{port:m}}function q(i,m){const{port:y}=M(i);if(!(y===void 0||y===m))return y}async function W(){var k;if(e.chainId===null||!e.execId||!e.beaconId)return;e.starting=!0,e.startError=null,g();const i={ChainID:e.chainId,ExecID:e.execId,BeaconID:e.beaconId,Archive:e.archive};e.dataDir&&(i.DataDir=e.dataDir),e.jwtPath&&(i.JWTPath=e.jwtPath);const m=q(e.execHTTPPort,ee),y=q(e.beaconHTTPPort,te),x=q(e.execP2PPort,ne);m!==void 0&&(i.ExecHTTPPort=m),y!==void 0&&(i.BeaconHTTPPort=y),x!==void 0&&(i.ExecP2PPort=x);try{await Te(e.targetId,i)}catch(S){if(!(S instanceof oe&&S.status===409)){e.starting=!1,e.startError=String(S instanceof Error?S.message:S),g();return}}e.starting=!1,e.step="run",e.events=[],g(),(k=e.streamStop)==null||k.call(e),e.streamStop=Pe(e.targetId,S=>{a||(e.events.push(S),e.step==="run"&&g())})}function O(i){const m=[{id:"network",label:"Network"},{id:"clients",label:"Clients"},{id:"mode",label:"Mode"},{id:"review",label:"Review"},{id:"run",label:"Run"}],x=m.map(k=>k.id).indexOf(i);return`
      <ol class="wizard-progress">
        ${m.map((k,S)=>`<li class="${S===x?"current":S<x?"past":"future"}">${o(k.label)}</li>`).join("")}
      </ol>
    `}return()=>{var i;a=!0,(i=e.streamStop)==null||i.call(e)}}const Qe=document.querySelector("#app"),{contentEl:Ze,setActiveNav:et}=Ae(Qe);let j=null;function tt(){const n=location.hash.replace(/^#\/?/,"").split("/").filter(Boolean);if(n.length===0)return{screen:"targets"};const[a,e]=n;return a==="setup"||a==="dash"||a==="logs"||a==="security"?{screen:a,id:e?decodeURIComponent(e):void 0}:{screen:a??"targets"}}function G(t){const n=document.createElement("div");return Ze.replaceChildren(n),t(n)}function ve(){if(j){try{j()}catch{}j=null}const{screen:t,id:n}=tt();switch(et(t),t){case"setup":if(!n){location.hash="#/targets";return}j=G(a=>Ye(a,n));break;case"dash":if(!n){location.hash="#/targets";return}j=G(a=>qe(a,n));break;case"logs":if(!n){location.hash="#/targets";return}j=G(a=>Oe(a,n));break;case"security":if(!n){location.hash="#/targets";return}j=G(a=>je(a,n));break;case"settings":j=G(a=>_e(a));break;case"targets":default:j=G(a=>Je(a));break}}window.addEventListener("hashchange",ve);ve();
