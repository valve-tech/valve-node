var ye=Object.defineProperty;var $e=(t,n,a)=>n in t?ye(t,n,{enumerable:!0,configurable:!0,writable:!0,value:a}):t[n]=a;var ie=(t,n,a)=>$e(t,typeof n!="symbol"?n+"":n,a);(function(){const n=document.createElement("link").relList;if(n&&n.supports&&n.supports("modulepreload"))return;for(const l of document.querySelectorAll('link[rel="modulepreload"]'))e(l);new MutationObserver(l=>{for(const u of l)if(u.type==="childList")for(const T of u.addedNodes)T.tagName==="LINK"&&T.rel==="modulepreload"&&e(T)}).observe(document,{childList:!0,subtree:!0});function a(l){const u={};return l.integrity&&(u.integrity=l.integrity),l.referrerPolicy&&(u.referrerPolicy=l.referrerPolicy),l.crossOrigin==="use-credentials"?u.credentials="include":l.crossOrigin==="anonymous"?u.credentials="omit":u.credentials="same-origin",u}function e(l){if(l.ep)return;l.ep=!0;const u=a(l);fetch(l.href,u)}})();function V(){return D("/api/catalog")}function X(){return D("/api/targets")}function ce(t){return D("/api/targets",{method:"POST",headers:Z,body:JSON.stringify(t)})}function we(t){return D(`/api/targets/${encodeURIComponent(t)}`,{method:"DELETE"})}function Te(t,n){return D(`/api/targets/${encodeURIComponent(t)}/setup`,{method:"POST",headers:Z,body:JSON.stringify(n)})}function xe(t,n){const a=new EventSource(`/api/targets/${encodeURIComponent(t)}/setup/stream`);return a.onmessage=e=>{try{n(JSON.parse(e.data))}catch{}},()=>a.close()}function Pe(t,n){const a=new EventSource(`/api/targets/${encodeURIComponent(t)}/monitor/stream`);return a.onmessage=e=>{try{n(JSON.parse(e.data))}catch{}},()=>a.close()}function Se(t,n=200){return D(`/api/targets/${encodeURIComponent(t)}/logs?n=${n}`)}function Ee(t,n){const a=new EventSource(`/api/targets/${encodeURIComponent(t)}/logs/stream`);return a.onmessage=e=>{try{n(JSON.parse(e.data))}catch{}},()=>a.close()}function le(t,n){const a=n===void 0?{}:{lines:n};return D(`/api/targets/${encodeURIComponent(t)}/explain`,{method:"POST",headers:Z,body:JSON.stringify(a)})}function ke(t,n,a){return D(`/api/targets/${encodeURIComponent(t)}/services/${n}/${a}`,{method:"POST"})}function Ce(t,n){return D(`/api/targets/${encodeURIComponent(t)}/services/${n}/clear`,{method:"POST",headers:Z,body:JSON.stringify({Confirm:n})})}function Le(t){return D(`/api/targets/${encodeURIComponent(t)}/du`)}function Ie(t){return D(`/api/targets/${encodeURIComponent(t)}/endpoints`)}function He(t){return D(`/api/targets/${encodeURIComponent(t)}/firewall`)}function Re(t){return D(`/api/targets/${encodeURIComponent(t)}/diagnostics`)}function Be(t){return D(`/api/targets/${encodeURIComponent(t)}/diagnostics/latest`)}function De(){return D("/api/settings")}function Ae(t){return D("/api/settings",{method:"PUT",headers:Z,body:JSON.stringify(t)})}class se extends Error{constructor(a,e){super(e);ie(this,"status");this.name="ApiError",this.status=a}}const Z={"Content-Type":"application/json"};async function D(t,n){const a=await fetch(t,n);if(!a.ok){let l=a.statusText||`HTTP ${a.status}`;try{const u=await a.json();u&&typeof u.error=="string"&&u.error&&(l=u.error)}catch{}throw new se(a.status,l)}if(a.status===204)return;const e=await a.text();return e?JSON.parse(e):void 0}const Ne="https://learn.valve.city/rpc";function r(t){return t.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;").replace(/'/g,"&#39;")}function U(t,n){const a=t&&n?` <span class="footer-sep">·</span> <a href="${r(n)}" target="_blank" rel="noopener noreferrer">${r(t)}</a>`:"";return`
    <footer class="footer">
      <a href="${r(Ne)}" target="_blank" rel="noopener noreferrer">Learn how this works → learn.valve.city/rpc</a>${a}
    </footer>
  `}function Me(t){t.innerHTML=`
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
  `;const n=t.querySelector("#content"),a=Array.from(t.querySelectorAll("[data-nav]"));return{contentEl:n,setActiveNav:l=>{for(const u of a)u.classList.toggle("active",u.dataset.nav===l)}}}function W(t){return Number.isFinite(t)?t.toLocaleString("en-US"):"—"}function Ue(t){return Number.isFinite(t)?`${t.toFixed(1)}%`:"—"}function qe(t){if(!Number.isFinite(t)||t<0)return"—";if(t<60)return`~${Math.round(t)}s`;const n=Math.round(t/60),a=Math.floor(n/60),e=n%60;if(a===0)return`~${e}m`;if(a<48)return`~${a}h ${e}m`;const l=Math.floor(a/24),u=a%24;return`~${l}d ${u}h`}function B(t,n){return`<span class="badge badge-${n}">${r(t)}</span>`}function de(t){return`<span class="dot dot-${t}"></span>`}const ue=["B","KB","MB","GB","TB","PB"];function G(t){if(!Number.isFinite(t)||t<0)return"—";if(t===0)return"0 B";let n=t,a=0;for(;n>=1024&&a<ue.length-1;)n/=1024,a++;const e=n<10?2:n<100?1:0;return`${n.toFixed(e)} ${ue[a]}`}async function oe(t){try{return await navigator.clipboard.writeText(t),!0}catch{return!1}}function Y(t,n){t.addEventListener("click",a=>{const e=a.target.closest("[data-action]");if(!e||!t.contains(e))return;const l=e.dataset.action;l&&n(l,e,a)})}const Oe=85,ae={exec:"Execution",beacon:"Beacon"};function Fe(t,n){let a=!1,e=null,l=null,u=null,T=null,m=null,H=null,k=null,R=null;const p={exec:null,beacon:null};let $=null;t.innerHTML=`<h1>Dashboard: ${r(n)}</h1><div id="dash-body"><p class="muted">Loading…</p></div><div id="dash-footer">${U()}</div>`;const x=t.querySelector("#dash-body"),h=t.querySelector("#dash-footer");x.addEventListener("click",s=>{const f=s.target.closest("[data-action]");if(!f||!x.contains(f))return;const v=f.dataset.action;if(v==="svc-action"){const b=f.dataset.svc,I=f.dataset.kind;b&&I&&M(b,I)}else if(v==="open-clear"){const b=f.dataset.svc;b&&Q(b)}else if(v==="copy"){const b=f.dataset.copy;b&&j(f,b)}else v==="retry-du"?o():v==="retry-endpoints"&&i()}),c();async function c(){let s,f;try{const[b,I]=await Promise.all([X(),V()]);s=b.find(A=>A.id===n),f=I}catch(b){if(a)return;x.innerHTML=`<p class="error">Failed to load target: ${r(String(b))}</p>`;return}if(a)return;if(!s){x.innerHTML=`<p class="error">Target "${r(n)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!s.wire){x.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(n)}">Run the setup wizard →</a></p>`;return}const v=f==null?void 0:f.networks.find(b=>b.ChainID===s.wire.ChainID);v&&(h.innerHTML=U(v.Name,v.LearnURL)),x.innerHTML='<p class="muted">Connecting…</p>',e=Pe(n,b=>{a||(w(b),l=b,u=b,S())}),o(),i()}async function o(){H=null;try{m=await Le(n)}catch(s){m=null,H=String(s instanceof Error?s.message:s)}a||S()}async function i(){R=null;try{k=await Ie(n)}catch(s){k=null,R=String(s instanceof Error?s.message:s)}a||S()}function w(s){if(!l)return;const f=(new Date(s.at).getTime()-new Date(l.at).getTime())/1e3,v=s.execHead-l.execHead;if(f>0&&v>=0){const b=v/f;T=T===null?b:T*.7+b*.3}}function S(){if(!u)return;const s=u;x.innerHTML=`
      <div class="card-grid">
        ${q(s)}
        ${K(s)}
        ${O(s)}
        ${d(s)}
        ${g(s)}
        ${y()}
        ${L(s)}
      </div>
      <p class="muted small">Last updated ${r(new Date(s.at).toLocaleTimeString())}</p>
    `}function N(s){const v=s.refHead>0?s.refHead-s.execHead:null,b=v!==null&&v>0&&T&&T>0?qe(v/T):v!==null&&v<=0?"caught up":"—";return{lag:v,eta:b}}function q(s){const{lag:f,eta:v}=N(s);return`
      <div class="card">
        <h3>Execution sync</h3>
        <p>${s.execSyncing?B("syncing","warn"):B("synced","ok")}</p>
        <dl class="stat-list">
          <div><dt>Local head</dt><dd>${W(s.execHead)}</dd></div>
          <div><dt>Reference head</dt><dd>${f!==null?W(s.refHead):"unavailable"}</dd></div>
          <div><dt>Lag</dt><dd>${f!==null?W(Math.max(f,0))+" blocks":"—"}</dd></div>
          <div><dt>ETA</dt><dd>${v}</dd></div>
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
    `}function d(s){const f=s.diskUsedPct>=Oe;return`
      <div class="card ${f?"card-warn":""}">
        <h3>Disk</h3>
        <div class="meter"><div class="meter-fill ${f?"meter-warn":""}" style="width:${Math.min(s.diskUsedPct,100)}%"></div></div>
        <p>${Ue(s.diskUsedPct)} used</p>
      </div>
    `}function g(s){if(H)return`
        <div class="card card-warn">
          <h3>Storage</h3>
          <p class="error small">${r(H)}</p>
          <button class="btn btn-ghost" data-action="retry-du">Retry</button>
        </div>
      `;if(!m)return'<div class="card"><h3>Storage</h3><p class="muted">Loading…</p></div>';const f=m.ExpectedExecBytes>0?Math.min(m.ExecBytes/m.ExpectedExecBytes*100,100):0,v=m.ExpectedBeaconBytes>0?Math.min(m.BeaconBytes/m.ExpectedBeaconBytes*100,100):0,{lag:b,eta:I}=N(s),A=b!==null&&b>0&&T!==null&&T>0;return`
      <div class="card">
        <h3>Storage</h3>
        <p class="muted small">Estimate — varies by client and pruning.</p>
        <p class="muted small">Execution — ${G(m.ExecBytes)} of ~${G(m.ExpectedExecBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${f}%"></div></div>
        ${A?`<p class="muted small">Estimated time remaining: ${r(I)}</p>`:""}
        <p class="muted small">Beacon — ${G(m.BeaconBytes)} of ~${G(m.ExpectedBeaconBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${v}%"></div></div>
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
      `;if(!k)return'<div class="card"><h3>Endpoints</h3><p class="muted">Loading…</p></div>';const s=k,f=s.ExecReachable&&!s.ChainIDMatches?`<p class="error small">Exec responded, but its chain id doesn't match this target's wire config.</p>`:"",v=s.Access==="ssh"?`
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
        ${f}
        ${v}
      </div>
    `}function E(s,f){const v=ae[s],b=p[s],I=(A,ve,be)=>`<button class="btn btn-ghost" data-action="svc-action" data-svc="${s}" data-kind="${A}" ${b!==null||be?"disabled":""}>${b===A?C():r(ve)}</button>`;return`
      <div class="service-row">
        <span>${r(v)} ${f?B("active","ok"):B("down","bad")}</span>
        <div class="service-actions">
          ${I("start","Start",f)}
          ${I("stop","Stop",!f)}
          ${I("restart","Restart",!1)}
          <button class="btn btn-danger" data-action="open-clear" data-svc="${s}" ${b!==null?"disabled":""}>Clear…</button>
        </div>
      </div>
    `}function L(s){return`
      <div class="card">
        <h3>Services</h3>
        ${E("exec",s.execActive)}
        ${E("beacon",s.beaconActive)}
        ${$?`<p class="error small">${r($)}</p>`:""}
        <p class="card-links">
          <a href="#/logs/${encodeURIComponent(n)}">View logs →</a>
          <a href="#/security/${encodeURIComponent(n)}">Security →</a>
          <a href="#/diag/${encodeURIComponent(n)}">Diagnostics →</a>
        </p>
      </div>
    `}function C(){return'<span class="spinner" aria-label="working"></span>'}async function M(s,f){if(p[s]===null){p[s]=f,$=null,S();try{await ke(n,s,f)}catch(v){$=`${ae[s]} ${f} failed: ${v instanceof Error?v.message:String(v)}`}p[s]=null,a||S()}}async function j(s,f){const v=await oe(f),b=s.textContent;s.textContent=v?"Copied!":"Copy failed",setTimeout(()=>{a||(s.textContent=b)},1500)}function Q(s){const f=ae[s],v=m?G(s==="exec"?m.ExecBytes:m.BeaconBytes):"unknown (disk usage hasn't loaded)";_(`
        <h2>Clear ${r(f)} data</h2>
        <p class="error">
          This stops the ${r(f.toLowerCase())} service, deletes its chain data under the
          node's data directory (current size: ${r(v)}), and starts it again. A full
          resync is required afterward.
        </p>
        <p>Type <code>${r(s)}</code> to confirm.</p>
        <input type="text" id="clear-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="clear-confirm-btn" disabled>Clear and resync</button>
        </div>
      `,A=>{if(A==="cancel"){z();return}A==="confirm"&&P(s)});const b=document.getElementById("clear-confirm-input"),I=document.getElementById("clear-confirm-btn");b==null||b.addEventListener("input",()=>{I&&(I.disabled=b.value.trim()!==s)}),b==null||b.focus()}async function P(s){const f=document.getElementById("clear-confirm-btn");f&&(f.disabled=!0,f.textContent="Clearing…");try{await Ce(n,s),z(),o()}catch(v){const b=document.querySelector("#clear-modal .modal");if(b){const I=document.createElement("p");I.className="error small",I.textContent=`Clear failed: ${v instanceof Error?v.message:String(v)}`,b.appendChild(I)}f&&(f.disabled=!1,f.textContent="Clear and resync")}}function _(s,f){z();const v=document.createElement("div");v.className="modal-overlay",v.id="clear-modal",v.innerHTML=`<div class="modal">${s}</div>`,v.addEventListener("click",b=>{const I=b.target.closest("[data-modal-action]");I!=null&&I.dataset.modalAction&&f(I.dataset.modalAction),b.target===v&&f("cancel")}),document.body.appendChild(v)}function z(){var s;(s=document.getElementById("clear-modal"))==null||s.remove()}return()=>{a=!0,e==null||e(),z()}}const pe=500,fe="valve-node.explain-consent";function je(t,n){let a=!1,e=null;const l=[];t.innerHTML=`
    <h1>Logs: ${r(n)}</h1>
    <div id="logs-body"><p class="muted">Loading…</p></div>
    <div id="logs-footer">${U()}</div>
  `;const u=t.querySelector("#logs-body"),T=t.querySelector("#logs-footer");Y(t,c=>{c==="explain"&&R()}),m();async function m(){let c,o;try{const[w,S]=await Promise.all([X(),V()]);c=w.find(N=>N.id===n),o=S}catch(w){if(a)return;u.innerHTML=`<p class="error">Failed to load target: ${r(String(w))}</p>`;return}if(a)return;if(!c){u.innerHTML=`<p class="error">Target "${r(n)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!c.wire){u.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(n)}">Run the setup wizard →</a></p>`;return}const i=o==null?void 0:o.networks.find(w=>w.ChainID===c.wire.ChainID);i&&(T.innerHTML=U(i.Name,i.LearnURL));try{const w=await Se(n,200);if(a)return;l.push(...w)}catch(w){if(a)return;u.innerHTML=`<p class="error">Failed to load logs: ${r(String(w))}</p>`;return}H(),e=Ee(n,w=>{a||(l.push(w),l.length>pe&&l.splice(0,l.length-pe),H())})}function H(){const c=l.filter(i=>i.severity==="error"||i.severity==="critical");u.innerHTML=`
      <div class="logs-layout">
        <section class="logs-tail">
          <div class="logs-tail-head">
            <h2>Live tail</h2>
            <button class="btn" data-action="explain">Explain with AI</button>
          </div>
          <div class="log-lines">${l.map(k).join("")}</div>
        </section>
        <section class="logs-errors">
          <h2>Error feed ${B(String(c.length),c.length?"bad":"neutral")}</h2>
          <div class="log-lines">${c.length?c.slice().reverse().map(k).join(""):'<p class="muted">No errors seen yet.</p>'}</div>
        </section>
      </div>
    `;const o=u.querySelector(".log-lines");o&&(o.scrollTop=o.scrollHeight)}function k(c){const o=c.severity||"info",i=c.learnUrl?` <a href="${r(c.learnUrl)}" target="_blank" rel="noopener noreferrer">learn →</a>`:"";return`
      <div class="log-line log-${r(o)}">
        <span class="log-time">${r(new Date(c.at).toLocaleTimeString())}</span>
        <span class="log-unit">${r(c.unit)}</span>
        <span class="log-sev">${r(o)}</span>
        <span class="log-text">${r(c.line)}</span>
        ${c.explain?`<div class="log-explain">${r(c.explain)}${i}</div>`:""}
      </div>
    `}async function R(){const c=l.filter(i=>i.severity==="error"||i.severity==="critical").map(i=>i.line).slice(-40);if(!(localStorage.getItem(fe)==="1")){p(c);return}await $(c)}function p(c){const o=c.length?`<pre class="explain-excerpt">${c.map(i=>r(i)).join(`
`)}</pre>`:'<p class="muted">No recent error lines are loaded yet — the server will auto-select its own recent error/critical lines instead.</p>';x(`
      <h2>Send logs to your AI provider?</h2>
      <p>
        The excerpt below will be sent to the AI provider configured in
        <a href="#/settings">Settings</a> to generate a plain-English
        explanation. This happens every time you click "Explain with AI";
        this confirmation only shows once per browser.
      </p>
      ${o}
      <div class="modal-actions">
        <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        <button class="btn btn-primary" data-modal-action="proceed">Send to AI provider</button>
      </div>
    `,i=>{i==="proceed"?(localStorage.setItem(fe,"1"),h(),$(c)):h()})}async function $(c){x('<h2>Explain with AI</h2><p class="muted">Asking the AI provider…</p>',()=>{});try{const o=c.length?await le(n,c):await le(n);if(a)return;x(`
        <h2>Explanation</h2>
        <div class="explain-text">${r(o.text)}</div>
        <details class="advanced">
          <summary>What was sent</summary>
          <pre class="explain-excerpt">${o.sentExcerpt.map(i=>r(i)).join(`
`)||"(no log lines — general question only)"}</pre>
        </details>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,i=>{i==="close"&&h()})}catch(o){if(a)return;if(o instanceof se&&o.status===409){x(`
          <h2>No AI provider configured</h2>
          <p>Set a provider and key in <a href="#/settings">Settings</a>, then try again.</p>
          <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
        `,i=>{i==="close"&&h()});return}x(`
        <h2>Explain failed</h2>
        <p class="error">${r(o instanceof Error?o.message:String(o))}</p>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,i=>{i==="close"&&h()})}}function x(c,o){h();const i=document.createElement("div");i.className="modal-overlay",i.id="explain-modal",i.innerHTML=`<div class="modal">${c}</div>`,i.addEventListener("click",w=>{const S=w.target.closest("[data-modal-action]");S!=null&&S.dataset.modalAction&&o(S.dataset.modalAction),w.target===i&&o("cancel")}),document.body.appendChild(i)}function h(){var c;(c=document.getElementById("explain-modal"))==null||c.remove()}return()=>{a=!0,e==null||e(),h()}}function _e(t,n){let a=!1,e=null,l=null,u=!1,T=!1;t.innerHTML=`<h1>Network diagnostics: ${r(n)}</h1><div id="diag-body"><p class="muted">Loading…</p></div><div id="diag-footer">${U()}</div>`;const m=t.querySelector("#diag-body"),H=t.querySelector("#diag-footer");Y(t,(o,i)=>{var w;if(o==="run")R();else if(o==="toggle")(w=i.closest(".check-item"))==null||w.classList.toggle("expanded");else if(o==="copy"){const S=i.dataset.copy;S&&c(i,S)}}),k();async function k(){let o,i;try{const[S,N]=await Promise.all([X(),V()]);o=S.find(q=>q.id===n),i=N}catch(S){if(a)return;m.innerHTML=`<p class="error">Failed to load target: ${r(String(S))}</p>`;return}if(a)return;if(!o){m.innerHTML=`<p class="error">Target "${r(n)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!o.wire){m.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(n)}">Run the setup wizard →</a></p>`;return}const w=i==null?void 0:i.networks.find(S=>S.ChainID===o.wire.ChainID);w&&(H.innerHTML=U(w.Name,w.LearnURL));try{e=await Be(n),T=!0}catch(S){l=String(S instanceof Error?S.message:S)}a||p()}async function R(){u=!0,l=null,p();try{e=await Re(n),T=!0}catch(o){l=String(o instanceof Error?o.message:o)}u=!1,a||p()}function p(){m.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(n)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Checks run in order and stop at the first failure — the last item is where your node's
          network stack breaks. Diagnostics also run automatically when an error shows up in the
          logs or a connection fails (service down, zero peers); the latest result is shown here.
          All probes are read-only — nothing is ever changed automatically.
        </p>
        <button class="btn" data-action="run" ${u?"disabled":""}>${u?"Running…":"Run diagnostics"}</button>
      </div>
      ${l?`<p class="error">${r(l)}</p>`:""}
      ${$()}
    `}function $(){if(!T&&!l)return'<p class="muted">Loading…</p>';if(!e)return`<p class="muted">No diagnostics have run yet for this target. Run them now, or they'll run on their own the next time something goes wrong.</p>`;const o=new Date(e.at).toLocaleString(),i=e.failedId?`<p><strong>Failed at: ${r(x(e.failedId))}.</strong> <span class="muted small">Later checks were skipped — fix this first, then re-run.</span></p>`:"<p><strong>All checks passed.</strong></p>";return`
      <p class="muted small">Last run ${r(o)} — trigger: ${r(e.trigger)}</p>
      ${i}
      <ul class="check-list">${e.items.map(h).join("")}</ul>
    `}function x(o){var i;return((i=e==null?void 0:e.items.find(w=>w.ID===o))==null?void 0:i.Title)??o}function h(o){const i=o.Status==="pass"?"ok":o.Status==="fail"?"bad":o.Status==="warn"?"warn":"neutral",w=o.ID===(e==null?void 0:e.failedId);return`
      <li class="check-item${w?" expanded":""}">
        <button class="check-head" data-action="toggle" type="button">
          ${B(w?"failed here":o.Status,i)}
          <strong>${r(o.Title)}</strong>
          <span class="muted small check-detail-inline">${r(o.Detail)}</span>
        </button>
        <div class="check-body">
          <details${w?" open":""}>
            <summary>Why this matters</summary>
            <p class="muted small">${r(o.Why)}</p>
          </details>
          ${o.Fix?`
                <details${w?" open":""}>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${r(o.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${r(o.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function c(o,i){const w=await oe(i),S=o.textContent;o.textContent=w?"Copied!":"Copy failed",setTimeout(()=>{a||(o.textContent=S)},1500)}return()=>{a=!0}}function ze(t,n){let a=!1,e=[],l=null,u=!1,T=!1;t.innerHTML=`<h1>Security: ${r(n)}</h1><div id="sec-body"><p class="muted">Loading…</p></div><div id="sec-footer">${U()}</div>`;const m=t.querySelector("#sec-body"),H=t.querySelector("#sec-footer");Y(t,(h,c)=>{var o;if(h==="rerun")R();else if(h==="toggle")(o=c.closest(".check-item"))==null||o.classList.toggle("expanded");else if(h==="copy"){const i=c.dataset.copy;i&&x(c,i)}}),k();async function k(){let h,c;try{const[i,w]=await Promise.all([X(),V()]);h=i.find(S=>S.id===n),c=w}catch(i){if(a)return;m.innerHTML=`<p class="error">Failed to load target: ${r(String(i))}</p>`;return}if(a)return;if(!h){m.innerHTML=`<p class="error">Target "${r(n)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!h.wire){m.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(n)}">Run the setup wizard →</a></p>`;return}const o=c==null?void 0:c.networks.find(i=>i.ChainID===h.wire.ChainID);o&&(H.innerHTML=U(o.Name,o.LearnURL)),await R()}async function R(){u=!0,l=null,p();try{e=await He(n),T=!0}catch(h){l=String(h instanceof Error?h.message:h)}u=!1,a||p()}function p(){m.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(n)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Every check here is a live, read-only probe run on the target — nothing is ever changed
          automatically. Each "Fix" is a copy-paste command for you to review and run yourself.
        </p>
        <button class="btn" data-action="rerun" ${u?"disabled":""}>${u?"Re-running…":"Re-run checks"}</button>
      </div>
      ${l?`<p class="error">${r(l)}</p>`:""}
      ${!T&&u?'<p class="muted">Loading…</p>':e.length?`<ul class="check-list">${e.map($).join("")}</ul>`:T?'<p class="muted">No checks returned.</p>':""}
    `}function $(h){const c=h.Status==="pass"?"ok":h.Status==="fail"?"bad":h.Status==="warn"?"warn":"neutral";return`
      <li class="check-item">
        <button class="check-head" data-action="toggle" type="button">
          ${B(h.Status,c)}
          <strong>${r(h.Title)}</strong>
          <span class="muted small check-detail-inline">${r(h.Detail)}</span>
        </button>
        <div class="check-body">
          <details>
            <summary>Why this matters</summary>
            <p class="muted small">${r(h.Why)}</p>
          </details>
          ${h.Fix?`
                <details open>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${r(h.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${r(h.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function x(h,c){const o=await oe(c),i=h.textContent;h.textContent=o?"Copied!":"Copy failed",setTimeout(()=>{a||(h.textContent=i)},1500)}return()=>{a=!0}}const We=[{value:"",label:"None"},{value:"gemini",label:"Gemini"},{value:"groq",label:"Groq"},{value:"ollama",label:"Ollama"}];function Je(t){let n=!1,a=!1,e=!1,l=null,u=!1,T=null;t.innerHTML=`<h1>Settings</h1><div id="settings-body"><p class="muted">Loading…</p></div>${U()}`;const m=t.querySelector("#settings-body");Y(t,p=>{if(p==="save"&&R(),p==="clear-key"){if(!T)return;a=!0;const $=t.querySelector("#ai-key");$&&($.value=""),k(T)}}),H();async function H(){try{const p=await De();if(n)return;T=p,k(p)}catch(p){if(n)return;m.innerHTML=`<p class="error">Failed to load settings: ${r(String(p))}</p>`}}function k(p){var h,c;const $=We.map(o=>`<option value="${o.value}" ${p.aiProvider===o.value?"selected":""}>${r(o.label)}</option>`).join("");m.innerHTML=`
      <form class="card" id="settings-form" onsubmit="return false">
        <label>
          AI provider
          <select id="ai-provider">${$}</select>
        </label>
        <label>
          API key
          <input id="ai-key" type="password" placeholder="${p.aiKeySet?"•••••••• (leave blank to keep)":"no key set"}" autocomplete="off" />
        </label>
        ${p.aiKeySet?'<button class="btn btn-ghost" type="button" data-action="clear-key">Clear saved key</button>':""}
        <p class="muted small">Keys stay on this machine — they're written to ~/.valve-node/config.json (mode 0600) and only sent to the provider you pick, never anywhere else.</p>
        <details class="advanced">
          <summary>Advanced</summary>
          <label>
            Reference RPC base
            <input id="ref-rpc-base" type="text" value="${r(p.refRpcBase)}" />
          </label>
          <p class="muted small">Used to compute head-lag on the dashboard. Leave the default unless you have your own reference endpoint.</p>
        </details>
        ${l?`<p class="error">${r(l)}</p>`:""}
        ${u?'<p class="ok">Saved.</p>':""}
        <button class="btn btn-primary" type="button" data-action="save" ${e?"disabled":""}>${e?"Saving…":"Save"}</button>
      </form>
    `;const x=t.querySelector("#ai-key");x==null||x.addEventListener("input",()=>{a=!0,u=!1}),(h=t.querySelector("#ai-provider"))==null||h.addEventListener("change",()=>{u=!1}),(c=t.querySelector("#ref-rpc-base"))==null||c.addEventListener("input",()=>{u=!1})}async function R(){const p=t.querySelector("#ai-provider"),$=t.querySelector("#ai-key"),x=t.querySelector("#ref-rpc-base");if(!p||!$||!x||!T)return;const h={aiProvider:p.value,refRpcBase:x.value.trim()};a&&(h.aiKey=$.value),e=!0,l=null,u=!1,k(T);try{const c=await Ae(h);if(n)return;T=c,a=!1,e=!1,u=!0,k(c)}catch(c){if(n)return;e=!1,l=String(c instanceof Error?c.message:c),k(T)}}return()=>{n=!0}}const Ke="local";function Ge(t){let n=!1;t.innerHTML=`
    <h1>Targets</h1>
    <div id="targets-body"><p class="muted">Loading…</p></div>
    ${U()}
  `;const a=t.querySelector("#targets-body");Y(t,(p,$)=>{u(p,$)}),e();async function e(){try{const[p,$]=await Promise.all([X(),V()]);if(n)return;l(p,$)}catch(p){if(n)return;a.innerHTML=`<p class="error">Failed to load targets: ${r(String(p))}</p>`}}function l(p,$){const x=p.find(i=>i.mode==="local"),h=p.filter(i=>i.mode==="ssh"),c=x?he(x,$):`
        <div class="card">
          <h2>This machine</h2>
          ${Ye()}
          <button class="btn" data-action="add-local">Add this machine as a target</button>
        </div>
      `,o=h.length?h.map(i=>he(i,$)).join(""):'<p class="muted">No SSH targets yet.</p>';a.innerHTML=`
      <section class="section">${c}</section>
      <section class="section">
        <h2>Servers over SSH</h2>
        <div class="card-grid">${o}</div>
        ${Ve()}
      </section>
    `}async function u(p,$){if(p==="add-local"){await T();return}if(p==="delete-target"){const x=$.dataset.id;if(!x||!confirm(`Remove target "${x}"? This does not touch anything already running on it.`))return;await m(x);return}p==="add-ssh"&&await H()}async function T(){R();try{await ce({id:Ke,mode:"local"}),await e()}catch(p){k(p)}}async function m(p){try{await we(p),await e()}catch($){k($)}}async function H(){const p=t.querySelector("#ssh-host"),$=t.querySelector("#ssh-user"),x=t.querySelector("#ssh-key"),h=t.querySelector("#ssh-port"),c=t.querySelector("#ssh-id");if(!p||!$||!x||!h||!c)return;const o=p.value.trim(),i=$.value.trim(),w=x.value.trim(),S=h.value.trim(),N=c.value.trim();if(R(),!o||!i||!w){k(new Error("host, user, and key path are required"));return}const q=N||Qe(o),K={Host:o,User:i,KeyPath:w};if(S){const d=Number.parseInt(S,10);if(!Number.isFinite(d)||d<=0){k(new Error("port must be a positive number"));return}K.Port=d}const O=t.querySelector("#ssh-submit");O&&(O.disabled=!0,O.textContent="Connecting…");try{await ce({id:q,mode:"ssh",ssh:K}),await e()}catch(d){k(d),O&&(O.disabled=!1,O.textContent="Add server")}}function k(p){let $=t.querySelector("#targets-error");$||(a.insertAdjacentHTML("afterbegin",'<p id="targets-error" class="error"></p>'),$=t.querySelector("#targets-error")),$.textContent=String(p instanceof Error?p.message:p)}function R(){var p;(p=t.querySelector("#targets-error"))==null||p.remove()}return()=>{n=!0}}function he(t,n){const a=t.wire,e=t.mode==="local"?"this machine":"SSH",l=t.mode==="ssh"&&t.ssh?`${r(t.ssh.User)}@${r(t.ssh.Host)}`:e;let u,T;if(!a)u=B("not set up","neutral"),T=`<a class="btn" href="#/setup/${encodeURIComponent(t.id)}">Run setup wizard</a>`;else{const m=n.networks.find(k=>k.ChainID===a.ChainID),H=m?m.Name:`chain ${a.ChainID}`;u=`${B(H,"ok")} ${B(a.ExecID,"neutral")} ${B(a.BeaconID,"neutral")}${a.Archive?" "+B("archive","warn"):""}`,T=`
      <a class="btn" href="#/dash/${encodeURIComponent(t.id)}">Dashboard</a>
      <a class="btn" href="#/logs/${encodeURIComponent(t.id)}">Logs</a>
      <a class="btn btn-ghost" href="#/setup/${encodeURIComponent(t.id)}">Re-run setup</a>
    `}return`
    <div class="card">
      <h2>${r(t.id)}</h2>
      <p class="muted">${l}</p>
      <p>${u}</p>
      <div class="card-actions">
        ${T}
        <button class="btn btn-danger" data-action="delete-target" data-id="${r(t.id)}">Remove</button>
      </div>
    </div>
  `}function Ve(){return`
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
  `}function Xe(){const t=navigator.userAgentData,n=(t==null?void 0:t.platform)||navigator.platform||navigator.userAgent;return/mac|win/i.test(n)&&!/linux|android/i.test(n)}function Ye(){return Xe()?`
      <p class="banner banner-warn">
        macOS and Windows are not supported node hosts — use this machine as a controller and add a
        Linux server over SSH.
      </p>
    `:'<p class="muted">The machine running valve-node. Setup only works on a Linux target.</p>'}function Qe(t){return t.toLowerCase().replace(/[^a-z0-9]+/g,"-").replace(/^-+|-+$/g,"")||"target"}const re=[{id:"preflight",title:"Preflight checks"},{id:"toolchain",title:"Ensure git + build toolchains"},{id:"install-exec",title:"Install execution client"},{id:"install-beacon",title:"Install beacon client"},{id:"wire",title:"Write JWT secret and systemd units"},{id:"start",title:"Start execution and beacon services"},{id:"handshake",title:"Verify execution/beacon handshake"}],ee=8545,te=5052,ne=30303,Ze=[369,943,1],me={369:"default",943:"practise here first"};function et(t,n){let a=!1;const e={targetId:n,step:"network",catalog:null,loadError:null,chainId:369,execId:null,beaconId:null,archive:!1,dataDir:"",jwtPath:"",execHTTPPort:"",beaconHTTPPort:"",execP2PPort:"",execHTTPPortError:null,beaconHTTPPortError:null,execP2PPortError:null,starting:!1,startError:null,events:[],streamStop:null};t.innerHTML=`<h1>Setup: ${r(n)}</h1><div id="wizard-body"><p class="muted">Loading catalog…</p></div><div id="wizard-footer">${U()}</div>`;const l=t.querySelector("#wizard-body"),u=t.querySelector("#wizard-footer");Y(t,(d,g)=>{o(d,g)}),T();async function T(){try{const[d,g]=await Promise.all([V(),X()]);if(a)return;e.catalog=d;const y=g.find(E=>E.id===n);y!=null&&y.wire&&(e.chainId=y.wire.ChainID,e.execId=y.wire.ExecID,e.beaconId=y.wire.BeaconID,e.archive=y.wire.Archive,y.wire.ExecHTTPPort&&(e.execHTTPPort=String(y.wire.ExecHTTPPort)),y.wire.BeaconHTTPPort&&(e.beaconHTTPPort=String(y.wire.BeaconHTTPPort)),y.wire.ExecP2PPort&&(e.execP2PPort=String(y.wire.ExecP2PPort))),m()}catch(d){if(a)return;e.loadError=String(d instanceof Error?d.message:d),m()}}function m(){if(e.loadError){l.innerHTML=`<p class="error">Failed to load: ${r(e.loadError)}</p>`;return}e.catalog&&(l.innerHTML=`
      ${O(e.step)}
      ${k()}
    `,H())}function H(){var g;const d=(g=e.catalog)==null?void 0:g.networks.find(y=>y.ChainID===e.chainId);u.innerHTML=d?U(d.Name,d.LearnURL):U()}function k(){switch(e.step){case"network":return R();case"clients":return p();case"mode":return x();case"review":return h();case"run":return c()}}function R(){const d=e.catalog;return`
      <section>
        <h2>1. Choose a network</h2>
        <div class="card-grid">${Ze.map(y=>{const E=d.networks.find(M=>M.ChainID===y);if(!E)return"";const L=e.chainId===y,C=me[y]?B(me[y],y===369?"ok":"warn"):"";return`
        <button class="card card-selectable ${L?"selected":""}" data-action="pick-network" data-chain-id="${y}" type="button">
          <h3>${r(E.Name)} <span class="muted">(chain ${y})</span></h3>
          ${C}
          <p class="muted small">Checkpoint sync from ${r(E.CheckpointURL)}</p>
        </button>
      `}).join("")}</div>
        <div class="wizard-actions">
          <button class="btn" data-action="goto-clients" ${e.chainId===null?"disabled":""}>Next: clients</button>
        </div>
      </section>
    `}function p(){const d=e.catalog,g=d.networks.find(L=>L.ChainID===e.chainId);if(!g)return'<p class="error">Unknown network.</p>';(e.execId===null||!g.ExecClients.includes(e.execId))&&(e.execId=g.ExecClients[0]??null),(e.beaconId===null||!g.BeaconClients.includes(e.beaconId))&&(e.beaconId=g.BeaconClients[0]??null);const y=g.ExecClients.map(L=>$(L,d,e.execId)).join(""),E=g.BeaconClients.map(L=>$(L,d,e.beaconId)).join("");return`
      <section>
        <h2>2. Choose your client pair</h2>
        <p class="muted">Only combinations known to work on ${r(g.Name)} are offered.</p>
        <label>
          Execution client
          <select id="exec-select" data-action="pick-exec">${y}</select>
        </label>
        <label>
          Beacon client
          <select id="beacon-select" data-action="pick-beacon">${E}</select>
        </label>
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-network">Back</button>
          <button class="btn" data-action="goto-mode">Next: mode</button>
        </div>
      </section>
    `}function $(d,g,y){const E=g.clients.find(C=>C.id===d),L=E?`${E.id} (${E.toolchain})`:d;return`<option value="${r(d)}" ${d===y?"selected":""}>${r(L)}</option>`}function x(){const d=e.chainId!==null?`/var/lib/valve-node/${e.chainId}`:"";return`
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
    `}function h(){const g=e.catalog.networks.find(P=>P.ChainID===e.chainId),y=e.dataDir||`/var/lib/valve-node/${e.chainId}`,E=e.jwtPath||`${y}/jwt.hex`,L=re.map(P=>`<li>${r(P.title)}</li>`).join(""),C=q(e.execHTTPPort,ee),M=q(e.beaconHTTPPort,te),j=q(e.execP2PPort,ne),Q=C||M||j?`<tr><th>Non-default ports</th><td>${[C?`exec HTTP ${C}`:null,M?`beacon HTTP ${M}`:null,j?`exec p2p ${j}`:null].filter(P=>P!==null).map(r).join(", ")}</td></tr>`:"";return`
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
            <tr><th>JWT secret path</th><td><code>${r(E)}</code></td></tr>
            ${Q}
          </tbody>
        </table>
        <p class="muted small">
          There is no preview API for the exact files/units that will be
          written — the list below is the fixed step sequence setup always
          runs; the actual commands and file contents stream live once you
          start.
        </p>
        <ol class="step-preview">${L}</ol>
        ${e.startError?`<p class="error">${r(e.startError)}</p>`:""}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-mode">Back</button>
          <button class="btn btn-primary" data-action="start-setup" ${e.starting?"disabled":""}>
            ${e.starting?"Starting…":"Start setup"}
          </button>
        </div>
      </section>
    `}function c(){const g=e.catalog.networks.find(P=>P.ChainID===e.chainId),y=g==null?void 0:g.LearnURL,E=new Set(e.events.filter(P=>P.done).map(P=>P.stepId)),L=new Set(e.events.filter(P=>P.err).map(P=>P.stepId)),C=new Map;for(const P of e.events){if(!P.line)continue;const _=C.get(P.stepId)??[];_.push(P.line),C.set(P.stepId,_)}const M=re.map(P=>{var I;const _=E.has(P.id),z=L.has(P.id),s=z?B("failed","bad"):_?B("done","ok"):B("pending","neutral"),f=(C.get(P.id)??[]).slice(-5),v=(I=e.events.find(A=>A.stepId===P.id&&A.err))==null?void 0:I.err,b=P.id==="handshake"?`<p class="muted small">"Talking" means the beacon client can reach the execution client's Engine API over the shared JWT secret and both report the same head — the sign your node is wired correctly.${y?` <a href="${r(y)}" target="_blank" rel="noopener noreferrer">Learn more →</a>`:""}</p>`:"";return`
        <li class="step-row ${_?"step-done":""} ${z?"step-error":""}">
          <div class="step-head">${s} <strong>${r(P.title)}</strong></div>
          ${b}
          ${f.length?`<pre class="step-log">${f.map(A=>r(A)).join(`
`)}</pre>`:""}
          ${v?`<p class="error small">${r(v)}</p>`:""}
        </li>
      `}).join(""),j=e.events.some(P=>P.err),Q=re.every(P=>E.has(P.id))||e.events.some(P=>P.stepId==="handshake"&&P.done);return`
      <section>
        <h2>5. Running setup</h2>
        <ol class="step-list">${M}</ol>
        ${Q&&!j?`<p class="ok">Setup complete. <a href="#/dash/${encodeURIComponent(e.targetId)}">Open the dashboard →</a></p>`:""}
        ${j?'<button class="btn" data-action="start-setup">Retry setup</button>':""}
      </section>
    `}function o(d,g){switch(d){case"pick-network":e.chainId=Number(g.dataset.chainId),e.execId=null,e.beaconId=null,m();break;case"goto-network":e.step="network",m();break;case"goto-clients":if(e.chainId===null)return;e.step="clients",m();break;case"goto-mode":i(),e.step="mode",m();break;case"goto-review":if(w(),e.execHTTPPortError||e.beaconHTTPPortError||e.execP2PPortError){m();break}e.step="review",m();break;case"start-setup":K();break}}function i(){const d=t.querySelector("#exec-select"),g=t.querySelector("#beacon-select");d&&(e.execId=d.value),g&&(e.beaconId=g.value)}function w(){const d=t.querySelectorAll('input[name="mode"]');for(const M of Array.from(d))M.checked&&(e.archive=M.value==="archive");const g=t.querySelector("#data-dir-input"),y=t.querySelector("#jwt-path-input");g&&(e.dataDir=g.value.trim()),y&&(e.jwtPath=y.value.trim());const E=t.querySelector("#exec-http-port-input"),L=t.querySelector("#beacon-http-port-input"),C=t.querySelector("#exec-p2p-port-input");E&&(e.execHTTPPort=E.value.trim()),L&&(e.beaconHTTPPort=L.value.trim()),C&&(e.execP2PPort=C.value.trim()),e.execHTTPPortError=N(e.execHTTPPort).error??null,e.beaconHTTPPortError=N(e.beaconHTTPPort).error??null,e.execP2PPortError=N(e.execP2PPort).error??null}const S=/^\d+$/;function N(d){if(!d)return{};if(!S.test(d))return{error:"Enter a whole number (no decimals, signs, or other characters)."};const g=Number(d);return!Number.isInteger(g)||g<1||g>65535?{error:"Port must be between 1 and 65535."}:{port:g}}function q(d,g){const{port:y}=N(d);if(!(y===void 0||y===g))return y}async function K(){var L;if(e.chainId===null||!e.execId||!e.beaconId)return;e.starting=!0,e.startError=null,m();const d={ChainID:e.chainId,ExecID:e.execId,BeaconID:e.beaconId,Archive:e.archive};e.dataDir&&(d.DataDir=e.dataDir),e.jwtPath&&(d.JWTPath=e.jwtPath);const g=q(e.execHTTPPort,ee),y=q(e.beaconHTTPPort,te),E=q(e.execP2PPort,ne);g!==void 0&&(d.ExecHTTPPort=g),y!==void 0&&(d.BeaconHTTPPort=y),E!==void 0&&(d.ExecP2PPort=E);try{await Te(e.targetId,d)}catch(C){if(!(C instanceof se&&C.status===409)){e.starting=!1,e.startError=String(C instanceof Error?C.message:C),m();return}}e.starting=!1,e.step="run",e.events=[],m(),(L=e.streamStop)==null||L.call(e),e.streamStop=xe(e.targetId,C=>{a||(e.events.push(C),e.step==="run"&&m())})}function O(d){const g=[{id:"network",label:"Network"},{id:"clients",label:"Clients"},{id:"mode",label:"Mode"},{id:"review",label:"Review"},{id:"run",label:"Run"}],E=g.map(L=>L.id).indexOf(d);return`
      <ol class="wizard-progress">
        ${g.map((L,C)=>`<li class="${C===E?"current":C<E?"past":"future"}">${r(L.label)}</li>`).join("")}
      </ol>
    `}return()=>{var d;a=!0,(d=e.streamStop)==null||d.call(e)}}const tt=document.querySelector("#app"),{contentEl:nt,setActiveNav:at}=Me(tt);let F=null;function rt(){const n=location.hash.replace(/^#\/?/,"").split("/").filter(Boolean);if(n.length===0)return{screen:"targets"};const[a,e]=n;return a==="setup"||a==="dash"||a==="logs"||a==="security"||a==="diag"?{screen:a,id:e?decodeURIComponent(e):void 0}:{screen:a??"targets"}}function J(t){const n=document.createElement("div");return nt.replaceChildren(n),t(n)}function ge(){if(F){try{F()}catch{}F=null}const{screen:t,id:n}=rt();switch(at(t),t){case"setup":if(!n){location.hash="#/targets";return}F=J(a=>et(a,n));break;case"dash":if(!n){location.hash="#/targets";return}F=J(a=>Fe(a,n));break;case"logs":if(!n){location.hash="#/targets";return}F=J(a=>je(a,n));break;case"security":if(!n){location.hash="#/targets";return}F=J(a=>ze(a,n));break;case"diag":if(!n){location.hash="#/targets";return}F=J(a=>_e(a,n));break;case"settings":F=J(a=>Je(a));break;case"targets":default:F=J(a=>Ge(a));break}}window.addEventListener("hashchange",ge);ge();
