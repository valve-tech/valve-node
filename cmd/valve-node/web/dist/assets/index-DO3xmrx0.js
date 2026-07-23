var ye=Object.defineProperty;var $e=(t,n,a)=>n in t?ye(t,n,{enumerable:!0,configurable:!0,writable:!0,value:a}):t[n]=a;var ie=(t,n,a)=>$e(t,typeof n!="symbol"?n+"":n,a);(function(){const n=document.createElement("link").relList;if(n&&n.supports&&n.supports("modulepreload"))return;for(const l of document.querySelectorAll('link[rel="modulepreload"]'))e(l);new MutationObserver(l=>{for(const p of l)if(p.type==="childList")for(const w of p.addedNodes)w.tagName==="LINK"&&w.rel==="modulepreload"&&e(w)}).observe(document,{childList:!0,subtree:!0});function a(l){const p={};return l.integrity&&(p.integrity=l.integrity),l.referrerPolicy&&(p.referrerPolicy=l.referrerPolicy),l.crossOrigin==="use-credentials"?p.credentials="include":l.crossOrigin==="anonymous"?p.credentials="omit":p.credentials="same-origin",p}function e(l){if(l.ep)return;l.ep=!0;const p=a(l);fetch(l.href,p)}})();function V(){return D("/api/catalog")}function X(){return D("/api/targets")}function ce(t){return D("/api/targets",{method:"POST",headers:Z,body:JSON.stringify(t)})}function we(t){return D(`/api/targets/${encodeURIComponent(t)}`,{method:"DELETE"})}function Te(t,n){return D(`/api/targets/${encodeURIComponent(t)}/setup`,{method:"POST",headers:Z,body:JSON.stringify(n)})}function xe(t,n){const a=new EventSource(`/api/targets/${encodeURIComponent(t)}/setup/stream`);return a.onmessage=e=>{try{n(JSON.parse(e.data))}catch{}},()=>a.close()}function Pe(t,n){const a=new EventSource(`/api/targets/${encodeURIComponent(t)}/monitor/stream`);return a.onmessage=e=>{try{n(JSON.parse(e.data))}catch{}},()=>a.close()}function Se(t,n=200){return D(`/api/targets/${encodeURIComponent(t)}/logs?n=${n}`)}function Ee(t,n){const a=new EventSource(`/api/targets/${encodeURIComponent(t)}/logs/stream`);return a.onmessage=e=>{try{n(JSON.parse(e.data))}catch{}},()=>a.close()}function le(t,n){const a=n===void 0?{}:{lines:n};return D(`/api/targets/${encodeURIComponent(t)}/explain`,{method:"POST",headers:Z,body:JSON.stringify(a)})}function ke(t,n,a){return D(`/api/targets/${encodeURIComponent(t)}/services/${n}/${a}`,{method:"POST"})}function Ce(t,n){return D(`/api/targets/${encodeURIComponent(t)}/services/${n}/clear`,{method:"POST",headers:Z,body:JSON.stringify({Confirm:n})})}function Le(t){return D(`/api/targets/${encodeURIComponent(t)}/du`)}function He(t){return D(`/api/targets/${encodeURIComponent(t)}/endpoints`)}function Ie(t){return D(`/api/targets/${encodeURIComponent(t)}/firewall`)}function Re(t){return D(`/api/targets/${encodeURIComponent(t)}/diagnostics`)}function Be(){return D("/api/settings")}function De(t){return D("/api/settings",{method:"PUT",headers:Z,body:JSON.stringify(t)})}class se extends Error{constructor(a,e){super(e);ie(this,"status");this.name="ApiError",this.status=a}}const Z={"Content-Type":"application/json"};async function D(t,n){const a=await fetch(t,n);if(!a.ok){let l=a.statusText||`HTTP ${a.status}`;try{const p=await a.json();p&&typeof p.error=="string"&&p.error&&(l=p.error)}catch{}throw new se(a.status,l)}if(a.status===204)return;const e=await a.text();return e?JSON.parse(e):void 0}const Ne="https://learn.valve.city/rpc";function r(t){return t.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;").replace(/'/g,"&#39;")}function M(t,n){const a=t&&n?` <span class="footer-sep">·</span> <a href="${r(n)}" target="_blank" rel="noopener noreferrer">${r(t)}</a>`:"";return`
    <footer class="footer">
      <a href="${r(Ne)}" target="_blank" rel="noopener noreferrer">Learn how this works → learn.valve.city/rpc</a>${a}
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
  `;const n=t.querySelector("#content"),a=Array.from(t.querySelectorAll("[data-nav]"));return{contentEl:n,setActiveNav:l=>{for(const p of a)p.classList.toggle("active",p.dataset.nav===l)}}}function W(t){return Number.isFinite(t)?t.toLocaleString("en-US"):"—"}function Me(t){return Number.isFinite(t)?`${t.toFixed(1)}%`:"—"}function Ue(t){if(!Number.isFinite(t)||t<0)return"—";if(t<60)return`~${Math.round(t)}s`;const n=Math.round(t/60),a=Math.floor(n/60),e=n%60;if(a===0)return`~${e}m`;if(a<48)return`~${a}h ${e}m`;const l=Math.floor(a/24),p=a%24;return`~${l}d ${p}h`}function B(t,n){return`<span class="badge badge-${n}">${r(t)}</span>`}function de(t){return`<span class="dot dot-${t}"></span>`}const ue=["B","KB","MB","GB","TB","PB"];function G(t){if(!Number.isFinite(t)||t<0)return"—";if(t===0)return"0 B";let n=t,a=0;for(;n>=1024&&a<ue.length-1;)n/=1024,a++;const e=n<10?2:n<100?1:0;return`${n.toFixed(e)} ${ue[a]}`}async function oe(t){try{return await navigator.clipboard.writeText(t),!0}catch{return!1}}function Y(t,n){t.addEventListener("click",a=>{const e=a.target.closest("[data-action]");if(!e||!t.contains(e))return;const l=e.dataset.action;l&&n(l,e,a)})}const qe=85,ae={exec:"Execution",beacon:"Beacon"};function Oe(t,n){let a=!1,e=null,l=null,p=null,w=null,m=null,I=null,S=null,R=null;const f={exec:null,beacon:null};let $=null;t.innerHTML=`<h1>Dashboard: ${r(n)}</h1><div id="dash-body"><p class="muted">Loading…</p></div><div id="dash-footer">${M()}</div>`;const T=t.querySelector("#dash-body"),i=t.querySelector("#dash-footer");T.addEventListener("click",s=>{const h=s.target.closest("[data-action]");if(!h||!T.contains(h))return;const b=h.dataset.action;if(b==="svc-action"){const v=h.dataset.svc,L=h.dataset.kind;v&&L&&A(v,L)}else if(b==="open-clear"){const v=h.dataset.svc;v&&Q(v)}else if(b==="copy"){const v=h.dataset.copy;v&&j(h,v)}else b==="retry-du"?u():b==="retry-endpoints"&&c()}),o();async function o(){let s,h;try{const[v,L]=await Promise.all([X(),V()]);s=v.find(N=>N.id===n),h=L}catch(v){if(a)return;T.innerHTML=`<p class="error">Failed to load target: ${r(String(v))}</p>`;return}if(a)return;if(!s){T.innerHTML=`<p class="error">Target "${r(n)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!s.wire){T.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(n)}">Run the setup wizard →</a></p>`;return}const b=h==null?void 0:h.networks.find(v=>v.ChainID===s.wire.ChainID);b&&(i.innerHTML=M(b.Name,b.LearnURL)),T.innerHTML='<p class="muted">Connecting…</p>',e=Pe(n,v=>{a||(k(v),l=v,p=v,H())}),u(),c()}async function u(){I=null;try{m=await Le(n)}catch(s){m=null,I=String(s instanceof Error?s.message:s)}a||H()}async function c(){R=null;try{S=await He(n)}catch(s){S=null,R=String(s instanceof Error?s.message:s)}a||H()}function k(s){if(!l)return;const h=(new Date(s.at).getTime()-new Date(l.at).getTime())/1e3,b=s.execHead-l.execHead;if(h>0&&b>=0){const v=b/h;w=w===null?v:w*.7+v*.3}}function H(){if(!p)return;const s=p;T.innerHTML=`
      <div class="card-grid">
        ${q(s)}
        ${K(s)}
        ${O(s)}
        ${d(s)}
        ${g(s)}
        ${y()}
        ${C(s)}
      </div>
      <p class="muted small">Last updated ${r(new Date(s.at).toLocaleTimeString())}</p>
    `}function U(s){const b=s.refHead>0?s.refHead-s.execHead:null,v=b!==null&&b>0&&w&&w>0?Ue(b/w):b!==null&&b<=0?"caught up":"—";return{lag:b,eta:v}}function q(s){const{lag:h,eta:b}=U(s);return`
      <div class="card">
        <h3>Execution sync</h3>
        <p>${s.execSyncing?B("syncing","warn"):B("synced","ok")}</p>
        <dl class="stat-list">
          <div><dt>Local head</dt><dd>${W(s.execHead)}</dd></div>
          <div><dt>Reference head</dt><dd>${h!==null?W(s.refHead):"unavailable"}</dd></div>
          <div><dt>Lag</dt><dd>${h!==null?W(Math.max(h,0))+" blocks":"—"}</dd></div>
          <div><dt>ETA</dt><dd>${b}</dd></div>
        </dl>
      </div>
    `}function K(s){return`
      <div class="card">
        <h3>Beacon sync</h3>
        <p>${s.beaconDistance===0?B("synced","ok"):B("syncing","warn")}</p>
        <dl class="stat-list">
          <div><dt>Slot</dt><dd>${W(s.beaconSlot)}</dd></div>
          <div><dt>Distance</dt><dd>${W(s.beaconDistance)}</dd></div>
        </dl>
      </div>
    `}function O(s){return`
      <div class="card">
        <h3>Peers</h3>
        <dl class="stat-list">
          <div><dt>Execution</dt><dd>${W(s.execPeers)}</dd></div>
          <div><dt>Beacon</dt><dd>${W(s.beaconPeers)}</dd></div>
        </dl>
      </div>
    `}function d(s){const h=s.diskUsedPct>=qe;return`
      <div class="card ${h?"card-warn":""}">
        <h3>Disk</h3>
        <div class="meter"><div class="meter-fill ${h?"meter-warn":""}" style="width:${Math.min(s.diskUsedPct,100)}%"></div></div>
        <p>${Me(s.diskUsedPct)} used</p>
      </div>
    `}function g(s){if(I)return`
        <div class="card card-warn">
          <h3>Storage</h3>
          <p class="error small">${r(I)}</p>
          <button class="btn btn-ghost" data-action="retry-du">Retry</button>
        </div>
      `;if(!m)return'<div class="card"><h3>Storage</h3><p class="muted">Loading…</p></div>';const h=m.ExpectedExecBytes>0?Math.min(m.ExecBytes/m.ExpectedExecBytes*100,100):0,b=m.ExpectedBeaconBytes>0?Math.min(m.BeaconBytes/m.ExpectedBeaconBytes*100,100):0,{lag:v,eta:L}=U(s),N=v!==null&&v>0&&w!==null&&w>0;return`
      <div class="card">
        <h3>Storage</h3>
        <p class="muted small">Estimate — varies by client and pruning.</p>
        <p class="muted small">Execution — ${G(m.ExecBytes)} of ~${G(m.ExpectedExecBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${h}%"></div></div>
        ${N?`<p class="muted small">Estimated time remaining: ${r(L)}</p>`:""}
        <p class="muted small">Beacon — ${G(m.BeaconBytes)} of ~${G(m.ExpectedBeaconBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${b}%"></div></div>
        <dl class="stat-list">
          <div><dt>Disk free</dt><dd>${G(m.DiskFreeBytes)}</dd></div>
          <div><dt>Sync (snapshot)</dt><dd>${r(m.SyncLabel)}</dd></div>
          <div><dt>Sync (genesis)</dt><dd>${r(m.GenesisSyncLabel)}</dd></div>
        </dl>
      </div>
    `}function y(){if(R)return`
        <div class="card card-warn">
          <h3>Endpoints</h3>
          <p class="error small">${r(R)}</p>
          <button class="btn btn-ghost" data-action="retry-endpoints">Retry</button>
        </div>
      `;if(!S)return'<div class="card"><h3>Endpoints</h3><p class="muted">Loading…</p></div>';const s=S,h=s.ExecReachable&&!s.ChainIDMatches?`<p class="error small">Exec responded, but its chain id doesn't match this target's wire config.</p>`:"",b=s.Access==="ssh"?`
          <p class="muted small">These URLs are local to the server; use the tunnel or your own reverse proxy to reach them from elsewhere.</p>
          <div class="endpoint-row">
            <code class="endpoint-url">${r(s.TunnelHint)}</code>
            <button class="btn btn-ghost" data-action="copy" data-copy="${r(s.TunnelHint)}">Copy</button>
          </div>
        `:"";return`
      <div class="card">
        <h3>Endpoints</h3>
        <div class="endpoint-row">
          ${de(s.ExecReachable?"ok":"bad")}
          <code class="endpoint-url">${r(s.ExecHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${r(s.ExecHTTP)}">Copy</button>
        </div>
        <div class="endpoint-row">
          ${de(s.BeaconReachable?"ok":"bad")}
          <code class="endpoint-url">${r(s.BeaconHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${r(s.BeaconHTTP)}">Copy</button>
        </div>
        ${h}
        ${b}
      </div>
    `}function P(s,h){const b=ae[s],v=f[s],L=(N,be,ve)=>`<button class="btn btn-ghost" data-action="svc-action" data-svc="${s}" data-kind="${N}" ${v!==null||ve?"disabled":""}>${v===N?E():r(be)}</button>`;return`
      <div class="service-row">
        <span>${r(b)} ${h?B("active","ok"):B("down","bad")}</span>
        <div class="service-actions">
          ${L("start","Start",h)}
          ${L("stop","Stop",!h)}
          ${L("restart","Restart",!1)}
          <button class="btn btn-danger" data-action="open-clear" data-svc="${s}" ${v!==null?"disabled":""}>Clear…</button>
        </div>
      </div>
    `}function C(s){return`
      <div class="card">
        <h3>Services</h3>
        ${P("exec",s.execActive)}
        ${P("beacon",s.beaconActive)}
        ${$?`<p class="error small">${r($)}</p>`:""}
        <p class="card-links">
          <a href="#/logs/${encodeURIComponent(n)}">View logs →</a>
          <a href="#/security/${encodeURIComponent(n)}">Security →</a>
          <a href="#/diag/${encodeURIComponent(n)}">Diagnostics →</a>
        </p>
      </div>
    `}function E(){return'<span class="spinner" aria-label="working"></span>'}async function A(s,h){if(f[s]===null){f[s]=h,$=null,H();try{await ke(n,s,h)}catch(b){$=`${ae[s]} ${h} failed: ${b instanceof Error?b.message:String(b)}`}f[s]=null,a||H()}}async function j(s,h){const b=await oe(h),v=s.textContent;s.textContent=b?"Copied!":"Copy failed",setTimeout(()=>{a||(s.textContent=v)},1500)}function Q(s){const h=ae[s],b=m?G(s==="exec"?m.ExecBytes:m.BeaconBytes):"unknown (disk usage hasn't loaded)";_(`
        <h2>Clear ${r(h)} data</h2>
        <p class="error">
          This stops the ${r(h.toLowerCase())} service, deletes its chain data under the
          node's data directory (current size: ${r(b)}), and starts it again. A full
          resync is required afterward.
        </p>
        <p>Type <code>${r(s)}</code> to confirm.</p>
        <input type="text" id="clear-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="clear-confirm-btn" disabled>Clear and resync</button>
        </div>
      `,N=>{if(N==="cancel"){z();return}N==="confirm"&&x(s)});const v=document.getElementById("clear-confirm-input"),L=document.getElementById("clear-confirm-btn");v==null||v.addEventListener("input",()=>{L&&(L.disabled=v.value.trim()!==s)}),v==null||v.focus()}async function x(s){const h=document.getElementById("clear-confirm-btn");h&&(h.disabled=!0,h.textContent="Clearing…");try{await Ce(n,s),z(),u()}catch(b){const v=document.querySelector("#clear-modal .modal");if(v){const L=document.createElement("p");L.className="error small",L.textContent=`Clear failed: ${b instanceof Error?b.message:String(b)}`,v.appendChild(L)}h&&(h.disabled=!1,h.textContent="Clear and resync")}}function _(s,h){z();const b=document.createElement("div");b.className="modal-overlay",b.id="clear-modal",b.innerHTML=`<div class="modal">${s}</div>`,b.addEventListener("click",v=>{const L=v.target.closest("[data-modal-action]");L!=null&&L.dataset.modalAction&&h(L.dataset.modalAction),v.target===b&&h("cancel")}),document.body.appendChild(b)}function z(){var s;(s=document.getElementById("clear-modal"))==null||s.remove()}return()=>{a=!0,e==null||e(),z()}}const pe=500,fe="valve-node.explain-consent";function Fe(t,n){let a=!1,e=null;const l=[];t.innerHTML=`
    <h1>Logs: ${r(n)}</h1>
    <div id="logs-body"><p class="muted">Loading…</p></div>
    <div id="logs-footer">${M()}</div>
  `;const p=t.querySelector("#logs-body"),w=t.querySelector("#logs-footer");Y(t,o=>{o==="explain"&&R()}),m();async function m(){let o,u;try{const[k,H]=await Promise.all([X(),V()]);o=k.find(U=>U.id===n),u=H}catch(k){if(a)return;p.innerHTML=`<p class="error">Failed to load target: ${r(String(k))}</p>`;return}if(a)return;if(!o){p.innerHTML=`<p class="error">Target "${r(n)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!o.wire){p.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(n)}">Run the setup wizard →</a></p>`;return}const c=u==null?void 0:u.networks.find(k=>k.ChainID===o.wire.ChainID);c&&(w.innerHTML=M(c.Name,c.LearnURL));try{const k=await Se(n,200);if(a)return;l.push(...k)}catch(k){if(a)return;p.innerHTML=`<p class="error">Failed to load logs: ${r(String(k))}</p>`;return}I(),e=Ee(n,k=>{a||(l.push(k),l.length>pe&&l.splice(0,l.length-pe),I())})}function I(){const o=l.filter(c=>c.severity==="error"||c.severity==="critical");p.innerHTML=`
      <div class="logs-layout">
        <section class="logs-tail">
          <div class="logs-tail-head">
            <h2>Live tail</h2>
            <button class="btn" data-action="explain">Explain with AI</button>
          </div>
          <div class="log-lines">${l.map(S).join("")}</div>
        </section>
        <section class="logs-errors">
          <h2>Error feed ${B(String(o.length),o.length?"bad":"neutral")}</h2>
          <div class="log-lines">${o.length?o.slice().reverse().map(S).join(""):'<p class="muted">No errors seen yet.</p>'}</div>
        </section>
      </div>
    `;const u=p.querySelector(".log-lines");u&&(u.scrollTop=u.scrollHeight)}function S(o){const u=o.severity||"info",c=o.learnUrl?` <a href="${r(o.learnUrl)}" target="_blank" rel="noopener noreferrer">learn →</a>`:"";return`
      <div class="log-line log-${r(u)}">
        <span class="log-time">${r(new Date(o.at).toLocaleTimeString())}</span>
        <span class="log-unit">${r(o.unit)}</span>
        <span class="log-sev">${r(u)}</span>
        <span class="log-text">${r(o.line)}</span>
        ${o.explain?`<div class="log-explain">${r(o.explain)}${c}</div>`:""}
      </div>
    `}async function R(){const o=l.filter(c=>c.severity==="error"||c.severity==="critical").map(c=>c.line).slice(-40);if(!(localStorage.getItem(fe)==="1")){f(o);return}await $(o)}function f(o){const u=o.length?`<pre class="explain-excerpt">${o.map(c=>r(c)).join(`
`)}</pre>`:'<p class="muted">No recent error lines are loaded yet — the server will auto-select its own recent error/critical lines instead.</p>';T(`
      <h2>Send logs to your AI provider?</h2>
      <p>
        The excerpt below will be sent to the AI provider configured in
        <a href="#/settings">Settings</a> to generate a plain-English
        explanation. This happens every time you click "Explain with AI";
        this confirmation only shows once per browser.
      </p>
      ${u}
      <div class="modal-actions">
        <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        <button class="btn btn-primary" data-modal-action="proceed">Send to AI provider</button>
      </div>
    `,c=>{c==="proceed"?(localStorage.setItem(fe,"1"),i(),$(o)):i()})}async function $(o){T('<h2>Explain with AI</h2><p class="muted">Asking the AI provider…</p>',()=>{});try{const u=o.length?await le(n,o):await le(n);if(a)return;T(`
        <h2>Explanation</h2>
        <div class="explain-text">${r(u.text)}</div>
        <details class="advanced">
          <summary>What was sent</summary>
          <pre class="explain-excerpt">${u.sentExcerpt.map(c=>r(c)).join(`
`)||"(no log lines — general question only)"}</pre>
        </details>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,c=>{c==="close"&&i()})}catch(u){if(a)return;if(u instanceof se&&u.status===409){T(`
          <h2>No AI provider configured</h2>
          <p>Set a provider and key in <a href="#/settings">Settings</a>, then try again.</p>
          <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
        `,c=>{c==="close"&&i()});return}T(`
        <h2>Explain failed</h2>
        <p class="error">${r(u instanceof Error?u.message:String(u))}</p>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,c=>{c==="close"&&i()})}}function T(o,u){i();const c=document.createElement("div");c.className="modal-overlay",c.id="explain-modal",c.innerHTML=`<div class="modal">${o}</div>`,c.addEventListener("click",k=>{const H=k.target.closest("[data-modal-action]");H!=null&&H.dataset.modalAction&&u(H.dataset.modalAction),k.target===c&&u("cancel")}),document.body.appendChild(c)}function i(){var o;(o=document.getElementById("explain-modal"))==null||o.remove()}return()=>{a=!0,e==null||e(),i()}}function je(t,n){let a=!1,e=[],l=null,p=!1,w=!1;t.innerHTML=`<h1>Network diagnostics: ${r(n)}</h1><div id="diag-body"><p class="muted">Loading…</p></div><div id="diag-footer">${M()}</div>`;const m=t.querySelector("#diag-body"),I=t.querySelector("#diag-footer");Y(t,(i,o)=>{var u;if(i==="rerun")R();else if(i==="toggle")(u=o.closest(".check-item"))==null||u.classList.toggle("expanded");else if(i==="copy"){const c=o.dataset.copy;c&&T(o,c)}}),S();async function S(){let i,o;try{const[c,k]=await Promise.all([X(),V()]);i=c.find(H=>H.id===n),o=k}catch(c){if(a)return;m.innerHTML=`<p class="error">Failed to load target: ${r(String(c))}</p>`;return}if(a)return;if(!i){m.innerHTML=`<p class="error">Target "${r(n)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!i.wire){m.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(n)}">Run the setup wizard →</a></p>`;return}const u=o==null?void 0:o.networks.find(c=>c.ChainID===i.wire.ChainID);u&&(I.innerHTML=M(u.Name,u.LearnURL)),await R()}async function R(){p=!0,l=null,f();try{e=await Re(n),w=!0}catch(i){l=String(i instanceof Error?i.message:i)}p=!1,a||f()}function f(){m.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(n)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Live, read-only probes run against the target — nothing is changed automatically.
          Run them when peers are low, sync is stuck, or you suspect a network problem; the
          checks are ordered so the first non-passing item is usually the root cause.
        </p>
        <button class="btn" data-action="rerun" ${p?"disabled":""}>${p?"Running…":"Run diagnostics"}</button>
      </div>
      ${l?`<p class="error">${r(l)}</p>`:""}
      ${!w&&p?'<p class="muted">Running probes…</p>':e.length?`<ul class="check-list">${e.map($).join("")}</ul>`:w?'<p class="muted">No checks returned.</p>':""}
    `}function $(i){const o=i.Status==="pass"?"ok":i.Status==="fail"?"bad":i.Status==="warn"?"warn":"neutral";return`
      <li class="check-item">
        <button class="check-head" data-action="toggle" type="button">
          ${B(i.Status,o)}
          <strong>${r(i.Title)}</strong>
          <span class="muted small check-detail-inline">${r(i.Detail)}</span>
        </button>
        <div class="check-body">
          <details>
            <summary>Why this matters</summary>
            <p class="muted small">${r(i.Why)}</p>
          </details>
          ${i.Fix?`
                <details open>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${r(i.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${r(i.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function T(i,o){const u=await oe(o),c=i.textContent;i.textContent=u?"Copied!":"Copy failed",setTimeout(()=>{a||(i.textContent=c)},1500)}return()=>{a=!0}}function _e(t,n){let a=!1,e=[],l=null,p=!1,w=!1;t.innerHTML=`<h1>Security: ${r(n)}</h1><div id="sec-body"><p class="muted">Loading…</p></div><div id="sec-footer">${M()}</div>`;const m=t.querySelector("#sec-body"),I=t.querySelector("#sec-footer");Y(t,(i,o)=>{var u;if(i==="rerun")R();else if(i==="toggle")(u=o.closest(".check-item"))==null||u.classList.toggle("expanded");else if(i==="copy"){const c=o.dataset.copy;c&&T(o,c)}}),S();async function S(){let i,o;try{const[c,k]=await Promise.all([X(),V()]);i=c.find(H=>H.id===n),o=k}catch(c){if(a)return;m.innerHTML=`<p class="error">Failed to load target: ${r(String(c))}</p>`;return}if(a)return;if(!i){m.innerHTML=`<p class="error">Target "${r(n)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!i.wire){m.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(n)}">Run the setup wizard →</a></p>`;return}const u=o==null?void 0:o.networks.find(c=>c.ChainID===i.wire.ChainID);u&&(I.innerHTML=M(u.Name,u.LearnURL)),await R()}async function R(){p=!0,l=null,f();try{e=await Ie(n),w=!0}catch(i){l=String(i instanceof Error?i.message:i)}p=!1,a||f()}function f(){m.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(n)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Every check here is a live, read-only probe run on the target — nothing is ever changed
          automatically. Each "Fix" is a copy-paste command for you to review and run yourself.
        </p>
        <button class="btn" data-action="rerun" ${p?"disabled":""}>${p?"Re-running…":"Re-run checks"}</button>
      </div>
      ${l?`<p class="error">${r(l)}</p>`:""}
      ${!w&&p?'<p class="muted">Loading…</p>':e.length?`<ul class="check-list">${e.map($).join("")}</ul>`:w?'<p class="muted">No checks returned.</p>':""}
    `}function $(i){const o=i.Status==="pass"?"ok":i.Status==="fail"?"bad":i.Status==="warn"?"warn":"neutral";return`
      <li class="check-item">
        <button class="check-head" data-action="toggle" type="button">
          ${B(i.Status,o)}
          <strong>${r(i.Title)}</strong>
          <span class="muted small check-detail-inline">${r(i.Detail)}</span>
        </button>
        <div class="check-body">
          <details>
            <summary>Why this matters</summary>
            <p class="muted small">${r(i.Why)}</p>
          </details>
          ${i.Fix?`
                <details open>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${r(i.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${r(i.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function T(i,o){const u=await oe(o),c=i.textContent;i.textContent=u?"Copied!":"Copy failed",setTimeout(()=>{a||(i.textContent=c)},1500)}return()=>{a=!0}}const ze=[{value:"",label:"None"},{value:"gemini",label:"Gemini"},{value:"groq",label:"Groq"},{value:"ollama",label:"Ollama"}];function We(t){let n=!1,a=!1,e=!1,l=null,p=!1,w=null;t.innerHTML=`<h1>Settings</h1><div id="settings-body"><p class="muted">Loading…</p></div>${M()}`;const m=t.querySelector("#settings-body");Y(t,f=>{if(f==="save"&&R(),f==="clear-key"){if(!w)return;a=!0;const $=t.querySelector("#ai-key");$&&($.value=""),S(w)}}),I();async function I(){try{const f=await Be();if(n)return;w=f,S(f)}catch(f){if(n)return;m.innerHTML=`<p class="error">Failed to load settings: ${r(String(f))}</p>`}}function S(f){var i,o;const $=ze.map(u=>`<option value="${u.value}" ${f.aiProvider===u.value?"selected":""}>${r(u.label)}</option>`).join("");m.innerHTML=`
      <form class="card" id="settings-form" onsubmit="return false">
        <label>
          AI provider
          <select id="ai-provider">${$}</select>
        </label>
        <label>
          API key
          <input id="ai-key" type="password" placeholder="${f.aiKeySet?"•••••••• (leave blank to keep)":"no key set"}" autocomplete="off" />
        </label>
        ${f.aiKeySet?'<button class="btn btn-ghost" type="button" data-action="clear-key">Clear saved key</button>':""}
        <p class="muted small">Keys stay on this machine — they're written to ~/.valve-node/config.json (mode 0600) and only sent to the provider you pick, never anywhere else.</p>
        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            Reference RPC base
            <input id="ref-rpc-base" type="text" value="${r(f.refRpcBase)}" />
          </label>
          <p class="muted small">Used to compute head-lag on the dashboard. Leave the default unless you have your own reference endpoint.</p>
        </details>
        ${l?`<p class="error">${r(l)}</p>`:""}
        ${p?'<p class="ok">Saved.</p>':""}
        <button class="btn btn-primary" type="button" data-action="save" ${e?"disabled":""}>${e?"Saving…":"Save"}</button>
      </form>
    `;const T=t.querySelector("#ai-key");T==null||T.addEventListener("input",()=>{a=!0,p=!1}),(i=t.querySelector("#ai-provider"))==null||i.addEventListener("change",()=>{p=!1}),(o=t.querySelector("#ref-rpc-base"))==null||o.addEventListener("input",()=>{p=!1})}async function R(){const f=t.querySelector("#ai-provider"),$=t.querySelector("#ai-key"),T=t.querySelector("#ref-rpc-base");if(!f||!$||!T||!w)return;const i={aiProvider:f.value,refRpcBase:T.value.trim()};a&&(i.aiKey=$.value),e=!0,l=null,p=!1,S(w);try{const o=await De(i);if(n)return;w=o,a=!1,e=!1,p=!0,S(o)}catch(o){if(n)return;e=!1,l=String(o instanceof Error?o.message:o),S(w)}}return()=>{n=!0}}const Je="local";function Ke(t){let n=!1;t.innerHTML=`
    <h1>Targets</h1>
    <div id="targets-body"><p class="muted">Loading…</p></div>
    ${M()}
  `;const a=t.querySelector("#targets-body");Y(t,(f,$)=>{p(f,$)}),e();async function e(){try{const[f,$]=await Promise.all([X(),V()]);if(n)return;l(f,$)}catch(f){if(n)return;a.innerHTML=`<p class="error">Failed to load targets: ${r(String(f))}</p>`}}function l(f,$){const T=f.find(c=>c.mode==="local"),i=f.filter(c=>c.mode==="ssh"),o=T?he(T,$):`
        <div class="card">
          <h2>This machine</h2>
          ${Xe()}
          <button class="btn" data-action="add-local">Add this machine as a target</button>
        </div>
      `,u=i.length?i.map(c=>he(c,$)).join(""):'<p class="muted">No SSH targets yet.</p>';a.innerHTML=`
      <section class="section">${o}</section>
      <section class="section">
        <h2>Servers over SSH</h2>
        <div class="card-grid">${u}</div>
        ${Ge()}
      </section>
    `}async function p(f,$){if(f==="add-local"){await w();return}if(f==="delete-target"){const T=$.dataset.id;if(!T||!confirm(`Remove target "${T}"? This does not touch anything already running on it.`))return;await m(T);return}f==="add-ssh"&&await I()}async function w(){R();try{await ce({id:Je,mode:"local"}),await e()}catch(f){S(f)}}async function m(f){try{await we(f),await e()}catch($){S($)}}async function I(){const f=t.querySelector("#ssh-host"),$=t.querySelector("#ssh-user"),T=t.querySelector("#ssh-key"),i=t.querySelector("#ssh-port"),o=t.querySelector("#ssh-id");if(!f||!$||!T||!i||!o)return;const u=f.value.trim(),c=$.value.trim(),k=T.value.trim(),H=i.value.trim(),U=o.value.trim();if(R(),!u||!c||!k){S(new Error("host, user, and key path are required"));return}const q=U||Ye(u),K={Host:u,User:c,KeyPath:k};if(H){const d=Number.parseInt(H,10);if(!Number.isFinite(d)||d<=0){S(new Error("port must be a positive number"));return}K.Port=d}const O=t.querySelector("#ssh-submit");O&&(O.disabled=!0,O.textContent="Connecting…");try{await ce({id:q,mode:"ssh",ssh:K}),await e()}catch(d){S(d),O&&(O.disabled=!1,O.textContent="Add server")}}function S(f){let $=t.querySelector("#targets-error");$||(a.insertAdjacentHTML("afterbegin",'<p id="targets-error" class="error"></p>'),$=t.querySelector("#targets-error")),$.textContent=String(f instanceof Error?f.message:f)}function R(){var f;(f=t.querySelector("#targets-error"))==null||f.remove()}return()=>{n=!0}}function he(t,n){const a=t.wire,e=t.mode==="local"?"this machine":"SSH",l=t.mode==="ssh"&&t.ssh?`${r(t.ssh.User)}@${r(t.ssh.Host)}`:e;let p,w;if(!a)p=B("not set up","neutral"),w=`<a class="btn" href="#/setup/${encodeURIComponent(t.id)}">Run setup wizard</a>`;else{const m=n.networks.find(S=>S.ChainID===a.ChainID),I=m?m.Name:`chain ${a.ChainID}`;p=`${B(I,"ok")} ${B(a.ExecID,"neutral")} ${B(a.BeaconID,"neutral")}${a.Archive?" "+B("archive","warn"):""}`,w=`
      <a class="btn" href="#/dash/${encodeURIComponent(t.id)}">Dashboard</a>
      <a class="btn" href="#/logs/${encodeURIComponent(t.id)}">Logs</a>
      <a class="btn btn-ghost" href="#/setup/${encodeURIComponent(t.id)}">Re-run setup</a>
    `}return`
    <div class="card">
      <h2>${r(t.id)}</h2>
      <p class="muted">${l}</p>
      <p>${p}</p>
      <div class="card-actions">
        ${w}
        <button class="btn btn-danger" data-action="delete-target" data-id="${r(t.id)}">Remove</button>
      </div>
    </div>
  `}function Ge(){return`
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
  `}function Ve(){const t=navigator.userAgentData,n=(t==null?void 0:t.platform)||navigator.platform||navigator.userAgent;return/mac|win/i.test(n)&&!/linux|android/i.test(n)}function Xe(){return Ve()?`
      <p class="banner banner-warn">
        macOS and Windows are not supported node hosts — use this machine as a controller and add a
        Linux server over SSH.
      </p>
    `:'<p class="muted">The machine running valve-node. Setup only works on a Linux target.</p>'}function Ye(t){return t.toLowerCase().replace(/[^a-z0-9]+/g,"-").replace(/^-+|-+$/g,"")||"target"}const re=[{id:"preflight",title:"Preflight checks"},{id:"toolchain",title:"Ensure git + build toolchains"},{id:"install-exec",title:"Install execution client"},{id:"install-beacon",title:"Install beacon client"},{id:"wire",title:"Write JWT secret and systemd units"},{id:"start",title:"Start execution and beacon services"},{id:"handshake",title:"Verify execution/beacon handshake"}],ee=8545,te=5052,ne=30303,Qe=[369,943,1],me={369:"default",943:"practise here first"};function Ze(t,n){let a=!1;const e={targetId:n,step:"network",catalog:null,loadError:null,chainId:369,execId:null,beaconId:null,archive:!1,dataDir:"",jwtPath:"",execHTTPPort:"",beaconHTTPPort:"",execP2PPort:"",execHTTPPortError:null,beaconHTTPPortError:null,execP2PPortError:null,starting:!1,startError:null,events:[],streamStop:null};t.innerHTML=`<h1>Setup: ${r(n)}</h1><div id="wizard-body"><p class="muted">Loading catalog…</p></div><div id="wizard-footer">${M()}</div>`;const l=t.querySelector("#wizard-body"),p=t.querySelector("#wizard-footer");Y(t,(d,g)=>{u(d,g)}),w();async function w(){try{const[d,g]=await Promise.all([V(),X()]);if(a)return;e.catalog=d;const y=g.find(P=>P.id===n);y!=null&&y.wire&&(e.chainId=y.wire.ChainID,e.execId=y.wire.ExecID,e.beaconId=y.wire.BeaconID,e.archive=y.wire.Archive,y.wire.ExecHTTPPort&&(e.execHTTPPort=String(y.wire.ExecHTTPPort)),y.wire.BeaconHTTPPort&&(e.beaconHTTPPort=String(y.wire.BeaconHTTPPort)),y.wire.ExecP2PPort&&(e.execP2PPort=String(y.wire.ExecP2PPort))),m()}catch(d){if(a)return;e.loadError=String(d instanceof Error?d.message:d),m()}}function m(){if(e.loadError){l.innerHTML=`<p class="error">Failed to load: ${r(e.loadError)}</p>`;return}e.catalog&&(l.innerHTML=`
      ${O(e.step)}
      ${S()}
    `,I())}function I(){var g;const d=(g=e.catalog)==null?void 0:g.networks.find(y=>y.ChainID===e.chainId);p.innerHTML=d?M(d.Name,d.LearnURL):M()}function S(){switch(e.step){case"network":return R();case"clients":return f();case"mode":return T();case"review":return i();case"run":return o()}}function R(){const d=e.catalog;return`
      <section>
        <h2>1. Choose a network</h2>
        <div class="card-grid">${Qe.map(y=>{const P=d.networks.find(A=>A.ChainID===y);if(!P)return"";const C=e.chainId===y,E=me[y]?B(me[y],y===369?"ok":"warn"):"";return`
        <button class="card card-selectable ${C?"selected":""}" data-action="pick-network" data-chain-id="${y}" type="button">
          <h3>${r(P.Name)} <span class="muted">(chain ${y})</span></h3>
          ${E}
          <p class="muted small">Checkpoint sync from ${r(P.CheckpointURL)}</p>
        </button>
      `}).join("")}</div>
        <div class="wizard-actions">
          <button class="btn" data-action="goto-clients" ${e.chainId===null?"disabled":""}>Next: clients</button>
        </div>
      </section>
    `}function f(){const d=e.catalog,g=d.networks.find(C=>C.ChainID===e.chainId);if(!g)return'<p class="error">Unknown network.</p>';(e.execId===null||!g.ExecClients.includes(e.execId))&&(e.execId=g.ExecClients[0]??null),(e.beaconId===null||!g.BeaconClients.includes(e.beaconId))&&(e.beaconId=g.BeaconClients[0]??null);const y=g.ExecClients.map(C=>$(C,d,e.execId)).join(""),P=g.BeaconClients.map(C=>$(C,d,e.beaconId)).join("");return`
      <section>
        <h2>2. Choose your client pair</h2>
        <p class="muted">Only combinations known to work on ${r(g.Name)} are offered.</p>
        <label>
          Execution client
          <select id="exec-select" data-action="pick-exec">${y}</select>
        </label>
        <label>
          Beacon client
          <select id="beacon-select" data-action="pick-beacon">${P}</select>
        </label>
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-network">Back</button>
          <button class="btn" data-action="goto-mode">Next: mode</button>
        </div>
      </section>
    `}function $(d,g,y){const P=g.clients.find(E=>E.id===d),C=P?`${P.id} (${P.toolchain})`:d;return`<option value="${r(d)}" ${d===y?"selected":""}>${r(C)}</option>`}function T(){const d=e.chainId!==null?`/var/lib/valve-node/${e.chainId}`:"";return`
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
            Data directory <span class="muted">(default: ${r(d)})</span>
            <input id="data-dir-input" type="text" placeholder="${r(d)}" value="${r(e.dataDir)}" />
          </label>
          <label>
            JWT secret path <span class="muted">(default: &lt;data dir&gt;/jwt.hex)</span>
            <input id="jwt-path-input" type="text" placeholder="${r(d)}/jwt.hex" value="${r(e.jwtPath)}" />
          </label>
          <label>
            Execution HTTP port <span class="muted">(default: ${ee})</span>
            <input id="exec-http-port-input" type="text" inputmode="numeric" placeholder="${ee}" value="${r(e.execHTTPPort)}" />
          </label>
          ${e.execHTTPPortError?`<p class="error small">${r(e.execHTTPPortError)}</p>`:""}
          <label>
            Beacon HTTP port <span class="muted">(default: ${te})</span>
            <input id="beacon-http-port-input" type="text" inputmode="numeric" placeholder="${te}" value="${r(e.beaconHTTPPort)}" />
          </label>
          ${e.beaconHTTPPortError?`<p class="error small">${r(e.beaconHTTPPortError)}</p>`:""}
          <label>
            Execution p2p port <span class="muted">(default: ${ne})</span>
            <input id="exec-p2p-port-input" type="text" inputmode="numeric" placeholder="${ne}" value="${r(e.execP2PPort)}" />
          </label>
          ${e.execP2PPortError?`<p class="error small">${r(e.execP2PPortError)}</p>`:""}
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
    `}function i(){const g=e.catalog.networks.find(x=>x.ChainID===e.chainId),y=e.dataDir||`/var/lib/valve-node/${e.chainId}`,P=e.jwtPath||`${y}/jwt.hex`,C=re.map(x=>`<li>${r(x.title)}</li>`).join(""),E=q(e.execHTTPPort,ee),A=q(e.beaconHTTPPort,te),j=q(e.execP2PPort,ne),Q=E||A||j?`<tr><th>Non-default ports</th><td>${[E?`exec HTTP ${E}`:null,A?`beacon HTTP ${A}`:null,j?`exec p2p ${j}`:null].filter(x=>x!==null).map(r).join(", ")}</td></tr>`:"";return`
      <section>
        <h2>4. Review</h2>
        <table class="review-table">
          <tbody>
            <tr><th>Target</th><td>${r(e.targetId)}</td></tr>
            <tr><th>Network</th><td>${r((g==null?void 0:g.Name)??String(e.chainId))} (chain ${e.chainId})</td></tr>
            <tr><th>Execution client</th><td>${r(e.execId??"")}</td></tr>
            <tr><th>Beacon client</th><td>${r(e.beaconId??"")}</td></tr>
            <tr><th>Mode</th><td>${e.archive?"Archive":"Full"}</td></tr>
            <tr><th>Data directory</th><td><code>${r(y)}</code></td></tr>
            <tr><th>JWT secret path</th><td><code>${r(P)}</code></td></tr>
            ${Q}
          </tbody>
        </table>
        <p class="muted small">
          There is no preview API for the exact files/units that will be
          written — the list below is the fixed step sequence setup always
          runs; the actual commands and file contents stream live once you
          start.
        </p>
        <ol class="step-preview">${C}</ol>
        ${e.startError?`<p class="error">${r(e.startError)}</p>`:""}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-mode">Back</button>
          <button class="btn btn-primary" data-action="start-setup" ${e.starting?"disabled":""}>
            ${e.starting?"Starting…":"Start setup"}
          </button>
        </div>
      </section>
    `}function o(){const g=e.catalog.networks.find(x=>x.ChainID===e.chainId),y=g==null?void 0:g.LearnURL,P=new Set(e.events.filter(x=>x.done).map(x=>x.stepId)),C=new Set(e.events.filter(x=>x.err).map(x=>x.stepId)),E=new Map;for(const x of e.events){if(!x.line)continue;const _=E.get(x.stepId)??[];_.push(x.line),E.set(x.stepId,_)}const A=re.map(x=>{var L;const _=P.has(x.id),z=C.has(x.id),s=z?B("failed","bad"):_?B("done","ok"):B("pending","neutral"),h=(E.get(x.id)??[]).slice(-5),b=(L=e.events.find(N=>N.stepId===x.id&&N.err))==null?void 0:L.err,v=x.id==="handshake"?`<p class="muted small">"Talking" means the beacon client can reach the execution client's Engine API over the shared JWT secret and both report the same head — the sign your node is wired correctly.${y?` <a href="${r(y)}" target="_blank" rel="noopener noreferrer">Learn more →</a>`:""}</p>`:"";return`
        <li class="step-row ${_?"step-done":""} ${z?"step-error":""}">
          <div class="step-head">${s} <strong>${r(x.title)}</strong></div>
          ${v}
          ${h.length?`<pre class="step-log">${h.map(N=>r(N)).join(`
`)}</pre>`:""}
          ${b?`<p class="error small">${r(b)}</p>`:""}
        </li>
      `}).join(""),j=e.events.some(x=>x.err),Q=re.every(x=>P.has(x.id))||e.events.some(x=>x.stepId==="handshake"&&x.done);return`
      <section>
        <h2>5. Running setup</h2>
        <ol class="step-list">${A}</ol>
        ${Q&&!j?`<p class="ok">Setup complete. <a href="#/dash/${encodeURIComponent(e.targetId)}">Open the dashboard →</a></p>`:""}
        ${j?'<button class="btn" data-action="start-setup">Retry setup</button>':""}
      </section>
    `}function u(d,g){switch(d){case"pick-network":e.chainId=Number(g.dataset.chainId),e.execId=null,e.beaconId=null,m();break;case"goto-network":e.step="network",m();break;case"goto-clients":if(e.chainId===null)return;e.step="clients",m();break;case"goto-mode":c(),e.step="mode",m();break;case"goto-review":if(k(),e.execHTTPPortError||e.beaconHTTPPortError||e.execP2PPortError){m();break}e.step="review",m();break;case"start-setup":K();break}}function c(){const d=t.querySelector("#exec-select"),g=t.querySelector("#beacon-select");d&&(e.execId=d.value),g&&(e.beaconId=g.value)}function k(){const d=t.querySelectorAll('input[name="mode"]');for(const A of Array.from(d))A.checked&&(e.archive=A.value==="archive");const g=t.querySelector("#data-dir-input"),y=t.querySelector("#jwt-path-input");g&&(e.dataDir=g.value.trim()),y&&(e.jwtPath=y.value.trim());const P=t.querySelector("#exec-http-port-input"),C=t.querySelector("#beacon-http-port-input"),E=t.querySelector("#exec-p2p-port-input");P&&(e.execHTTPPort=P.value.trim()),C&&(e.beaconHTTPPort=C.value.trim()),E&&(e.execP2PPort=E.value.trim()),e.execHTTPPortError=U(e.execHTTPPort).error??null,e.beaconHTTPPortError=U(e.beaconHTTPPort).error??null,e.execP2PPortError=U(e.execP2PPort).error??null}const H=/^\d+$/;function U(d){if(!d)return{};if(!H.test(d))return{error:"Enter a whole number (no decimals, signs, or other characters)."};const g=Number(d);return!Number.isInteger(g)||g<1||g>65535?{error:"Port must be between 1 and 65535."}:{port:g}}function q(d,g){const{port:y}=U(d);if(!(y===void 0||y===g))return y}async function K(){var C;if(e.chainId===null||!e.execId||!e.beaconId)return;e.starting=!0,e.startError=null,m();const d={ChainID:e.chainId,ExecID:e.execId,BeaconID:e.beaconId,Archive:e.archive};e.dataDir&&(d.DataDir=e.dataDir),e.jwtPath&&(d.JWTPath=e.jwtPath);const g=q(e.execHTTPPort,ee),y=q(e.beaconHTTPPort,te),P=q(e.execP2PPort,ne);g!==void 0&&(d.ExecHTTPPort=g),y!==void 0&&(d.BeaconHTTPPort=y),P!==void 0&&(d.ExecP2PPort=P);try{await Te(e.targetId,d)}catch(E){if(!(E instanceof se&&E.status===409)){e.starting=!1,e.startError=String(E instanceof Error?E.message:E),m();return}}e.starting=!1,e.step="run",e.events=[],m(),(C=e.streamStop)==null||C.call(e),e.streamStop=xe(e.targetId,E=>{a||(e.events.push(E),e.step==="run"&&m())})}function O(d){const g=[{id:"network",label:"Network"},{id:"clients",label:"Clients"},{id:"mode",label:"Mode"},{id:"review",label:"Review"},{id:"run",label:"Run"}],P=g.map(C=>C.id).indexOf(d);return`
      <ol class="wizard-progress">
        ${g.map((C,E)=>`<li class="${E===P?"current":E<P?"past":"future"}">${r(C.label)}</li>`).join("")}
      </ol>
    `}return()=>{var d;a=!0,(d=e.streamStop)==null||d.call(e)}}const et=document.querySelector("#app"),{contentEl:tt,setActiveNav:nt}=Ae(et);let F=null;function at(){const n=location.hash.replace(/^#\/?/,"").split("/").filter(Boolean);if(n.length===0)return{screen:"targets"};const[a,e]=n;return a==="setup"||a==="dash"||a==="logs"||a==="security"||a==="diag"?{screen:a,id:e?decodeURIComponent(e):void 0}:{screen:a??"targets"}}function J(t){const n=document.createElement("div");return tt.replaceChildren(n),t(n)}function ge(){if(F){try{F()}catch{}F=null}const{screen:t,id:n}=at();switch(nt(t),t){case"setup":if(!n){location.hash="#/targets";return}F=J(a=>Ze(a,n));break;case"dash":if(!n){location.hash="#/targets";return}F=J(a=>Oe(a,n));break;case"logs":if(!n){location.hash="#/targets";return}F=J(a=>Fe(a,n));break;case"security":if(!n){location.hash="#/targets";return}F=J(a=>_e(a,n));break;case"diag":if(!n){location.hash="#/targets";return}F=J(a=>je(a,n));break;case"settings":F=J(a=>We(a));break;case"targets":default:F=J(a=>Ke(a));break}}window.addEventListener("hashchange",ge);ge();
