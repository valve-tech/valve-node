var ge=Object.defineProperty;var ye=(t,n,a)=>n in t?ge(t,n,{enumerable:!0,configurable:!0,writable:!0,value:a}):t[n]=a;var se=(t,n,a)=>ye(t,typeof n!="symbol"?n+"":n,a);(function(){const n=document.createElement("link").relList;if(n&&n.supports&&n.supports("modulepreload"))return;for(const c of document.querySelectorAll('link[rel="modulepreload"]'))e(c);new MutationObserver(c=>{for(const u of c)if(u.type==="childList")for(const S of u.addedNodes)S.tagName==="LINK"&&S.rel==="modulepreload"&&e(S)}).observe(document,{childList:!0,subtree:!0});function a(c){const u={};return c.integrity&&(u.integrity=c.integrity),c.referrerPolicy&&(u.referrerPolicy=c.referrerPolicy),c.crossOrigin==="use-credentials"?u.credentials="include":c.crossOrigin==="anonymous"?u.credentials="omit":u.credentials="same-origin",u}function e(c){if(c.ep)return;c.ep=!0;const u=a(c);fetch(c.href,u)}})();function G(){return D("/api/catalog")}function V(){return D("/api/targets")}function oe(t){return D("/api/targets",{method:"POST",headers:X,body:JSON.stringify(t)})}function $e(t){return D(`/api/targets/${encodeURIComponent(t)}`,{method:"DELETE"})}function we(t,n){return D(`/api/targets/${encodeURIComponent(t)}/setup`,{method:"POST",headers:X,body:JSON.stringify(n)})}function xe(t,n){const a=new EventSource(`/api/targets/${encodeURIComponent(t)}/setup/stream`);return a.onmessage=e=>{try{n(JSON.parse(e.data))}catch{}},()=>a.close()}function Se(t,n){const a=new EventSource(`/api/targets/${encodeURIComponent(t)}/monitor/stream`);return a.onmessage=e=>{try{n(JSON.parse(e.data))}catch{}},()=>a.close()}function Te(t,n=200){return D(`/api/targets/${encodeURIComponent(t)}/logs?n=${n}`)}function Pe(t,n){const a=new EventSource(`/api/targets/${encodeURIComponent(t)}/logs/stream`);return a.onmessage=e=>{try{n(JSON.parse(e.data))}catch{}},()=>a.close()}function ie(t,n){const a=n===void 0?{}:{lines:n};return D(`/api/targets/${encodeURIComponent(t)}/explain`,{method:"POST",headers:X,body:JSON.stringify(a)})}function Ee(t,n,a){return D(`/api/targets/${encodeURIComponent(t)}/services/${n}/${a}`,{method:"POST"})}function ke(t,n){return D(`/api/targets/${encodeURIComponent(t)}/services/${n}/clear`,{method:"POST",headers:X,body:JSON.stringify({Confirm:n})})}function Ce(t){return D(`/api/targets/${encodeURIComponent(t)}/du`)}function Ie(t){return D(`/api/targets/${encodeURIComponent(t)}/endpoints`)}function Le(t){return D(`/api/targets/${encodeURIComponent(t)}/firewall`)}function He(){return D("/api/settings")}function Re(t){return D("/api/settings",{method:"PUT",headers:X,body:JSON.stringify(t)})}class re extends Error{constructor(a,e){super(e);se(this,"status");this.name="ApiError",this.status=a}}const X={"Content-Type":"application/json"};async function D(t,n){const a=await fetch(t,n);if(!a.ok){let c=a.statusText||`HTTP ${a.status}`;try{const u=await a.json();u&&typeof u.error=="string"&&u.error&&(c=u.error)}catch{}throw new re(a.status,c)}if(a.status===204)return;const e=await a.text();return e?JSON.parse(e):void 0}const Be="https://learn.valve.city/rpc";function s(t){return t.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;").replace(/'/g,"&#39;")}function N(t,n){const a=t&&n?` <span class="footer-sep">·</span> <a href="${s(n)}" target="_blank" rel="noopener noreferrer">${s(t)}</a>`:"";return`
    <footer class="footer">
      <a href="${s(Be)}" target="_blank" rel="noopener noreferrer">Learn how this works → learn.valve.city/rpc</a>${a}
    </footer>
  `}function De(t){t.innerHTML=`
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
  `;const n=t.querySelector("#content"),a=Array.from(t.querySelectorAll("[data-nav]"));return{contentEl:n,setActiveNav:c=>{for(const u of a)u.classList.toggle("active",u.dataset.nav===c)}}}function F(t){return Number.isFinite(t)?t.toLocaleString("en-US"):"—"}function Ae(t){return Number.isFinite(t)?`${t.toFixed(1)}%`:"—"}function Ne(t){if(!Number.isFinite(t)||t<0)return"—";if(t<60)return`~${Math.round(t)}s`;const n=Math.round(t/60),a=Math.floor(n/60),e=n%60;if(a===0)return`~${e}m`;if(a<48)return`~${a}h ${e}m`;const c=Math.floor(a/24),u=a%24;return`~${c}d ${u}h`}function R(t,n){return`<span class="badge badge-${n}">${s(t)}</span>`}function ce(t){return`<span class="dot dot-${t}"></span>`}const le=["B","KB","MB","GB","TB","PB"];function z(t){if(!Number.isFinite(t)||t<0)return"—";if(t===0)return"0 B";let n=t,a=0;for(;n>=1024&&a<le.length-1;)n/=1024,a++;const e=n<10?2:n<100?1:0;return`${n.toFixed(e)} ${le[a]}`}async function he(t){try{return await navigator.clipboard.writeText(t),!0}catch{return!1}}function Y(t,n){t.addEventListener("click",a=>{const e=a.target.closest("[data-action]");if(!e||!t.contains(e))return;const c=e.dataset.action;c&&n(c,e,a)})}const Me=85,ne={exec:"Execution",beacon:"Beacon"};function Ue(t,n){let a=!1,e=null,c=null,u=null,S=null,b=null,H=null,k=null,B=null;const d={exec:null,beacon:null};let $=null;t.innerHTML=`<h1>Dashboard: ${s(n)}</h1><div id="dash-body"><p class="muted">Loading…</p></div><div id="dash-footer">${N()}</div>`;const x=t.querySelector("#dash-body"),h=t.querySelector("#dash-footer");x.addEventListener("click",r=>{const f=r.target.closest("[data-action]");if(!f||!x.contains(f))return;const y=f.dataset.action;if(y==="svc-action"){const g=f.dataset.svc,C=f.dataset.kind;g&&C&&U(g,C)}else if(y==="open-clear"){const g=f.dataset.svc;g&&w(g)}else if(y==="copy"){const g=f.dataset.copy;g&&K(f,g)}else y==="retry-du"?m():y==="retry-endpoints"&&i()}),o();async function o(){let r,f;try{const[g,C]=await Promise.all([V(),G()]);r=g.find(j=>j.id===n),f=C}catch(g){if(a)return;x.innerHTML=`<p class="error">Failed to load target: ${s(String(g))}</p>`;return}if(a)return;if(!r){x.innerHTML=`<p class="error">Target "${s(n)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!r.wire){x.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(n)}">Run the setup wizard →</a></p>`;return}const y=f==null?void 0:f.networks.find(g=>g.ChainID===r.wire.ChainID);y&&(h.innerHTML=N(y.Name,y.LearnURL)),x.innerHTML='<p class="muted">Connecting…</p>',e=Se(n,g=>{a||(L(g),c=g,u=g,I())}),m(),i()}async function m(){H=null;try{b=await Ce(n)}catch(r){b=null,H=String(r instanceof Error?r.message:r)}a||I()}async function i(){B=null;try{k=await Ie(n)}catch(r){k=null,B=String(r instanceof Error?r.message:r)}a||I()}function L(r){if(!c)return;const f=(new Date(r.at).getTime()-new Date(c.at).getTime())/1e3,y=r.execHead-c.execHead;if(f>0&&y>=0){const g=y/f;S=S===null?g:S*.7+g*.3}}function I(){if(!u)return;const r=u;x.innerHTML=`
      <div class="card-grid">
        ${q(r)}
        ${W(r)}
        ${l(r)}
        ${v(r)}
        ${p()}
        ${T()}
        ${P(r)}
      </div>
      <p class="muted small">Last updated ${s(new Date(r.at).toLocaleTimeString())}</p>
    `}function q(r){const f=r.refHead>0,y=f?r.refHead-r.execHead:null,g=y!==null&&y>0&&S&&S>0?Ne(y/S):y!==null&&y<=0?"caught up":"—";return`
      <div class="card">
        <h3>Execution sync</h3>
        <p>${r.execSyncing?R("syncing","warn"):R("synced","ok")}</p>
        <dl class="stat-list">
          <div><dt>Local head</dt><dd>${F(r.execHead)}</dd></div>
          <div><dt>Reference head</dt><dd>${f?F(r.refHead):"unavailable"}</dd></div>
          <div><dt>Lag</dt><dd>${y!==null?F(Math.max(y,0))+" blocks":"—"}</dd></div>
          <div><dt>ETA</dt><dd>${g}</dd></div>
        </dl>
      </div>
    `}function W(r){return`
      <div class="card">
        <h3>Beacon sync</h3>
        <p>${r.beaconDistance===0?R("synced","ok"):R("syncing","warn")}</p>
        <dl class="stat-list">
          <div><dt>Slot</dt><dd>${F(r.beaconSlot)}</dd></div>
          <div><dt>Distance</dt><dd>${F(r.beaconDistance)}</dd></div>
        </dl>
      </div>
    `}function l(r){return`
      <div class="card">
        <h3>Peers</h3>
        <dl class="stat-list">
          <div><dt>Execution</dt><dd>${F(r.execPeers)}</dd></div>
          <div><dt>Beacon</dt><dd>${F(r.beaconPeers)}</dd></div>
        </dl>
      </div>
    `}function v(r){const f=r.diskUsedPct>=Me;return`
      <div class="card ${f?"card-warn":""}">
        <h3>Disk</h3>
        <div class="meter"><div class="meter-fill ${f?"meter-warn":""}" style="width:${Math.min(r.diskUsedPct,100)}%"></div></div>
        <p>${Ae(r.diskUsedPct)} used</p>
      </div>
    `}function p(){if(H)return`
        <div class="card card-warn">
          <h3>Storage</h3>
          <p class="error small">${s(H)}</p>
          <button class="btn btn-ghost" data-action="retry-du">Retry</button>
        </div>
      `;if(!b)return'<div class="card"><h3>Storage</h3><p class="muted">Loading…</p></div>';const r=b.ExpectedExecBytes>0?Math.min(b.ExecBytes/b.ExpectedExecBytes*100,100):0,f=b.ExpectedBeaconBytes>0?Math.min(b.BeaconBytes/b.ExpectedBeaconBytes*100,100):0;return`
      <div class="card">
        <h3>Storage</h3>
        <p class="muted small">Estimate — varies by client and pruning.</p>
        <p class="muted small">Execution — ${z(b.ExecBytes)} of ~${z(b.ExpectedExecBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${r}%"></div></div>
        <p class="muted small">Beacon — ${z(b.BeaconBytes)} of ~${z(b.ExpectedBeaconBytes)}</p>
        <div class="meter"><div class="meter-fill" style="width:${f}%"></div></div>
        <dl class="stat-list">
          <div><dt>Disk free</dt><dd>${z(b.DiskFreeBytes)}</dd></div>
          <div><dt>Sync (snapshot)</dt><dd>${s(b.SyncLabel)}</dd></div>
          <div><dt>Sync (genesis)</dt><dd>${s(b.GenesisSyncLabel)}</dd></div>
        </dl>
      </div>
    `}function T(){if(B)return`
        <div class="card card-warn">
          <h3>Endpoints</h3>
          <p class="error small">${s(B)}</p>
          <button class="btn btn-ghost" data-action="retry-endpoints">Retry</button>
        </div>
      `;if(!k)return'<div class="card"><h3>Endpoints</h3><p class="muted">Loading…</p></div>';const r=k,f=r.ExecReachable&&!r.ChainIDMatches?`<p class="error small">Exec responded, but its chain id doesn't match this target's wire config.</p>`:"",y=r.Access==="ssh"?`
          <p class="muted small">These URLs are local to the server; use the tunnel or your own reverse proxy to reach them from elsewhere.</p>
          <div class="endpoint-row">
            <code class="endpoint-url">${s(r.TunnelHint)}</code>
            <button class="btn btn-ghost" data-action="copy" data-copy="${s(r.TunnelHint)}">Copy</button>
          </div>
        `:"";return`
      <div class="card">
        <h3>Endpoints</h3>
        <div class="endpoint-row">
          ${ce(r.ExecReachable?"ok":"bad")}
          <code class="endpoint-url">${s(r.ExecHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${s(r.ExecHTTP)}">Copy</button>
        </div>
        <div class="endpoint-row">
          ${ce(r.BeaconReachable?"ok":"bad")}
          <code class="endpoint-url">${s(r.BeaconHTTP)}</code>
          <button class="btn btn-ghost" data-action="copy" data-copy="${s(r.BeaconHTTP)}">Copy</button>
        </div>
        ${f}
        ${y}
      </div>
    `}function E(r,f){const y=ne[r],g=d[r],C=(j,ve,be)=>`<button class="btn btn-ghost" data-action="svc-action" data-svc="${r}" data-kind="${j}" ${g!==null||be?"disabled":""}>${g===j?A():s(ve)}</button>`;return`
      <div class="service-row">
        <span>${s(y)} ${f?R("active","ok"):R("down","bad")}</span>
        <div class="service-actions">
          ${C("start","Start",f)}
          ${C("stop","Stop",!f)}
          ${C("restart","Restart",!1)}
          <button class="btn btn-danger" data-action="open-clear" data-svc="${r}" ${g!==null?"disabled":""}>Clear…</button>
        </div>
      </div>
    `}function P(r){return`
      <div class="card">
        <h3>Services</h3>
        ${E("exec",r.execActive)}
        ${E("beacon",r.beaconActive)}
        ${$?`<p class="error small">${s($)}</p>`:""}
        <p class="card-links">
          <a href="#/logs/${encodeURIComponent(n)}">View logs →</a>
          <a href="#/security/${encodeURIComponent(n)}">Security →</a>
        </p>
      </div>
    `}function A(){return'<span class="spinner" aria-label="working"></span>'}async function U(r,f){if(d[r]===null){d[r]=f,$=null,I();try{await Ee(n,r,f)}catch(y){$=`${ne[r]} ${f} failed: ${y instanceof Error?y.message:String(y)}`}d[r]=null,a||I()}}async function K(r,f){const y=await he(f),g=r.textContent;r.textContent=y?"Copied!":"Copy failed",setTimeout(()=>{a||(r.textContent=g)},1500)}function w(r){const f=ne[r],y=b?z(r==="exec"?b.ExecBytes:b.BeaconBytes):"unknown (disk usage hasn't loaded)";Q(`
        <h2>Clear ${s(f)} data</h2>
        <p class="error">
          This stops the ${s(f.toLowerCase())} service, deletes its chain data under the
          node's data directory (current size: ${s(y)}), and starts it again. A full
          resync is required afterward.
        </p>
        <p>Type <code>${s(r)}</code> to confirm.</p>
        <input type="text" id="clear-confirm-input" autocomplete="off" spellcheck="false" />
        <div class="modal-actions">
          <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
          <button class="btn btn-danger" data-modal-action="confirm" id="clear-confirm-btn" disabled>Clear and resync</button>
        </div>
      `,j=>{if(j==="cancel"){_();return}j==="confirm"&&O(r)});const g=document.getElementById("clear-confirm-input"),C=document.getElementById("clear-confirm-btn");g==null||g.addEventListener("input",()=>{C&&(C.disabled=g.value.trim()!==r)}),g==null||g.focus()}async function O(r){const f=document.getElementById("clear-confirm-btn");f&&(f.disabled=!0,f.textContent="Clearing…");try{await ke(n,r),_(),m()}catch(y){const g=document.querySelector("#clear-modal .modal");if(g){const C=document.createElement("p");C.className="error small",C.textContent=`Clear failed: ${y instanceof Error?y.message:String(y)}`,g.appendChild(C)}f&&(f.disabled=!1,f.textContent="Clear and resync")}}function Q(r,f){_();const y=document.createElement("div");y.className="modal-overlay",y.id="clear-modal",y.innerHTML=`<div class="modal">${r}</div>`,y.addEventListener("click",g=>{const C=g.target.closest("[data-modal-action]");C!=null&&C.dataset.modalAction&&f(C.dataset.modalAction),g.target===y&&f("cancel")}),document.body.appendChild(y)}function _(){var r;(r=document.getElementById("clear-modal"))==null||r.remove()}return()=>{a=!0,e==null||e(),_()}}const de=500,ue="valve-node.explain-consent";function qe(t,n){let a=!1,e=null;const c=[];t.innerHTML=`
    <h1>Logs: ${s(n)}</h1>
    <div id="logs-body"><p class="muted">Loading…</p></div>
    <div id="logs-footer">${N()}</div>
  `;const u=t.querySelector("#logs-body"),S=t.querySelector("#logs-footer");Y(t,o=>{o==="explain"&&B()}),b();async function b(){let o,m;try{const[L,I]=await Promise.all([V(),G()]);o=L.find(q=>q.id===n),m=I}catch(L){if(a)return;u.innerHTML=`<p class="error">Failed to load target: ${s(String(L))}</p>`;return}if(a)return;if(!o){u.innerHTML=`<p class="error">Target "${s(n)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!o.wire){u.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(n)}">Run the setup wizard →</a></p>`;return}const i=m==null?void 0:m.networks.find(L=>L.ChainID===o.wire.ChainID);i&&(S.innerHTML=N(i.Name,i.LearnURL));try{const L=await Te(n,200);if(a)return;c.push(...L)}catch(L){if(a)return;u.innerHTML=`<p class="error">Failed to load logs: ${s(String(L))}</p>`;return}H(),e=Pe(n,L=>{a||(c.push(L),c.length>de&&c.splice(0,c.length-de),H())})}function H(){const o=c.filter(i=>i.severity==="error"||i.severity==="critical");u.innerHTML=`
      <div class="logs-layout">
        <section class="logs-tail">
          <div class="logs-tail-head">
            <h2>Live tail</h2>
            <button class="btn" data-action="explain">Explain with AI</button>
          </div>
          <div class="log-lines">${c.map(k).join("")}</div>
        </section>
        <section class="logs-errors">
          <h2>Error feed ${R(String(o.length),o.length?"bad":"neutral")}</h2>
          <div class="log-lines">${o.length?o.slice().reverse().map(k).join(""):'<p class="muted">No errors seen yet.</p>'}</div>
        </section>
      </div>
    `;const m=u.querySelector(".log-lines");m&&(m.scrollTop=m.scrollHeight)}function k(o){const m=o.severity||"info",i=o.learnUrl?` <a href="${s(o.learnUrl)}" target="_blank" rel="noopener noreferrer">learn →</a>`:"";return`
      <div class="log-line log-${s(m)}">
        <span class="log-time">${s(new Date(o.at).toLocaleTimeString())}</span>
        <span class="log-unit">${s(o.unit)}</span>
        <span class="log-sev">${s(m)}</span>
        <span class="log-text">${s(o.line)}</span>
        ${o.explain?`<div class="log-explain">${s(o.explain)}${i}</div>`:""}
      </div>
    `}async function B(){const o=c.filter(i=>i.severity==="error"||i.severity==="critical").map(i=>i.line).slice(-40);if(!(localStorage.getItem(ue)==="1")){d(o);return}await $(o)}function d(o){const m=o.length?`<pre class="explain-excerpt">${o.map(i=>s(i)).join(`
`)}</pre>`:'<p class="muted">No recent error lines are loaded yet — the server will auto-select its own recent error/critical lines instead.</p>';x(`
      <h2>Send logs to your AI provider?</h2>
      <p>
        The excerpt below will be sent to the AI provider configured in
        <a href="#/settings">Settings</a> to generate a plain-English
        explanation. This happens every time you click "Explain with AI";
        this confirmation only shows once per browser.
      </p>
      ${m}
      <div class="modal-actions">
        <button class="btn btn-ghost" data-modal-action="cancel">Cancel</button>
        <button class="btn btn-primary" data-modal-action="proceed">Send to AI provider</button>
      </div>
    `,i=>{i==="proceed"?(localStorage.setItem(ue,"1"),h(),$(o)):h()})}async function $(o){x('<h2>Explain with AI</h2><p class="muted">Asking the AI provider…</p>',()=>{});try{const m=o.length?await ie(n,o):await ie(n);if(a)return;x(`
        <h2>Explanation</h2>
        <div class="explain-text">${s(m.text)}</div>
        <details class="advanced">
          <summary>What was sent</summary>
          <pre class="explain-excerpt">${m.sentExcerpt.map(i=>s(i)).join(`
`)||"(no log lines — general question only)"}</pre>
        </details>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,i=>{i==="close"&&h()})}catch(m){if(a)return;if(m instanceof re&&m.status===409){x(`
          <h2>No AI provider configured</h2>
          <p>Set a provider and key in <a href="#/settings">Settings</a>, then try again.</p>
          <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
        `,i=>{i==="close"&&h()});return}x(`
        <h2>Explain failed</h2>
        <p class="error">${s(m instanceof Error?m.message:String(m))}</p>
        <div class="modal-actions"><button class="btn" data-modal-action="close">Close</button></div>
      `,i=>{i==="close"&&h()})}}function x(o,m){h();const i=document.createElement("div");i.className="modal-overlay",i.id="explain-modal",i.innerHTML=`<div class="modal">${o}</div>`,i.addEventListener("click",L=>{const I=L.target.closest("[data-modal-action]");I!=null&&I.dataset.modalAction&&m(I.dataset.modalAction),L.target===i&&m("cancel")}),document.body.appendChild(i)}function h(){var o;(o=document.getElementById("explain-modal"))==null||o.remove()}return()=>{a=!0,e==null||e(),h()}}function Oe(t,n){let a=!1,e=[],c=null,u=!1,S=!1;t.innerHTML=`<h1>Security: ${s(n)}</h1><div id="sec-body"><p class="muted">Loading…</p></div><div id="sec-footer">${N()}</div>`;const b=t.querySelector("#sec-body"),H=t.querySelector("#sec-footer");Y(t,(h,o)=>{var m;if(h==="rerun")B();else if(h==="toggle")(m=o.closest(".check-item"))==null||m.classList.toggle("expanded");else if(h==="copy"){const i=o.dataset.copy;i&&x(o,i)}}),k();async function k(){let h,o;try{const[i,L]=await Promise.all([V(),G()]);h=i.find(I=>I.id===n),o=L}catch(i){if(a)return;b.innerHTML=`<p class="error">Failed to load target: ${s(String(i))}</p>`;return}if(a)return;if(!h){b.innerHTML=`<p class="error">Target "${s(n)}" not found. <a href="#/targets">Back to targets</a></p>`;return}if(!h.wire){b.innerHTML=`<p class="muted">This target hasn't completed setup yet. <a href="#/setup/${encodeURIComponent(n)}">Run the setup wizard →</a></p>`;return}const m=o==null?void 0:o.networks.find(i=>i.ChainID===h.wire.ChainID);m&&(H.innerHTML=N(m.Name,m.LearnURL)),await B()}async function B(){u=!0,c=null,d();try{e=await Le(n),S=!0}catch(h){c=String(h instanceof Error?h.message:h)}u=!1,a||d()}function d(){b.innerHTML=`
      <p><a href="#/dash/${encodeURIComponent(n)}">← Back to dashboard</a></p>
      <div class="section-head">
        <p class="muted small">
          Every check here is a live, read-only probe run on the target — nothing is ever changed
          automatically. Each "Fix" is a copy-paste command for you to review and run yourself.
        </p>
        <button class="btn" data-action="rerun" ${u?"disabled":""}>${u?"Re-running…":"Re-run checks"}</button>
      </div>
      ${c?`<p class="error">${s(c)}</p>`:""}
      ${!S&&u?'<p class="muted">Loading…</p>':e.length?`<ul class="check-list">${e.map($).join("")}</ul>`:S?'<p class="muted">No checks returned.</p>':""}
    `}function $(h){const o=h.Status==="pass"?"ok":h.Status==="fail"?"bad":h.Status==="warn"?"warn":"neutral";return`
      <li class="check-item">
        <button class="check-head" data-action="toggle" type="button">
          ${R(h.Status,o)}
          <strong>${s(h.Title)}</strong>
          <span class="muted small check-detail-inline">${s(h.Detail)}</span>
        </button>
        <div class="check-body">
          <details>
            <summary>Why this matters</summary>
            <p class="muted small">${s(h.Why)}</p>
          </details>
          ${h.Fix?`
                <details open>
                  <summary>Suggested fix</summary>
                  <pre class="fix-block">${s(h.Fix)}</pre>
                  <button class="btn btn-ghost" data-action="copy" data-copy="${s(h.Fix)}">Copy</button>
                </details>
              `:""}
        </div>
      </li>
    `}async function x(h,o){const m=await he(o),i=h.textContent;h.textContent=m?"Copied!":"Copy failed",setTimeout(()=>{a||(h.textContent=i)},1500)}return()=>{a=!0}}const je=[{value:"",label:"None"},{value:"gemini",label:"Gemini"},{value:"groq",label:"Groq"},{value:"ollama",label:"Ollama"}];function Fe(t){let n=!1,a=!1,e=!1,c=null,u=!1,S=null;t.innerHTML=`<h1>Settings</h1><div id="settings-body"><p class="muted">Loading…</p></div>${N()}`;const b=t.querySelector("#settings-body");Y(t,d=>{if(d==="save"&&B(),d==="clear-key"){if(!S)return;a=!0;const $=t.querySelector("#ai-key");$&&($.value=""),k(S)}}),H();async function H(){try{const d=await He();if(n)return;S=d,k(d)}catch(d){if(n)return;b.innerHTML=`<p class="error">Failed to load settings: ${s(String(d))}</p>`}}function k(d){var h,o;const $=je.map(m=>`<option value="${m.value}" ${d.aiProvider===m.value?"selected":""}>${s(m.label)}</option>`).join("");b.innerHTML=`
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
            <input id="ref-rpc-base" type="text" value="${s(d.refRpcBase)}" />
          </label>
          <p class="muted small">Used to compute head-lag on the dashboard. Leave the default unless you have your own reference endpoint.</p>
        </details>
        ${c?`<p class="error">${s(c)}</p>`:""}
        ${u?'<p class="ok">Saved.</p>':""}
        <button class="btn btn-primary" type="button" data-action="save" ${e?"disabled":""}>${e?"Saving…":"Save"}</button>
      </form>
    `;const x=t.querySelector("#ai-key");x==null||x.addEventListener("input",()=>{a=!0,u=!1}),(h=t.querySelector("#ai-provider"))==null||h.addEventListener("change",()=>{u=!1}),(o=t.querySelector("#ref-rpc-base"))==null||o.addEventListener("input",()=>{u=!1})}async function B(){const d=t.querySelector("#ai-provider"),$=t.querySelector("#ai-key"),x=t.querySelector("#ref-rpc-base");if(!d||!$||!x||!S)return;const h={aiProvider:d.value,refRpcBase:x.value.trim()};a&&(h.aiKey=$.value),e=!0,c=null,u=!1,k(S);try{const o=await Re(h);if(n)return;S=o,a=!1,e=!1,u=!0,k(o)}catch(o){if(n)return;e=!1,c=String(o instanceof Error?o.message:o),k(S)}}return()=>{n=!0}}const _e="local";function ze(t){let n=!1;t.innerHTML=`
    <h1>Targets</h1>
    <div id="targets-body"><p class="muted">Loading…</p></div>
    ${N()}
  `;const a=t.querySelector("#targets-body");Y(t,(d,$)=>{u(d,$)}),e();async function e(){try{const[d,$]=await Promise.all([V(),G()]);if(n)return;c(d,$)}catch(d){if(n)return;a.innerHTML=`<p class="error">Failed to load targets: ${s(String(d))}</p>`}}function c(d,$){const x=d.find(i=>i.mode==="local"),h=d.filter(i=>i.mode==="ssh"),o=x?pe(x,$):`
        <div class="card">
          <h2>This machine</h2>
          ${Ke()}
          <button class="btn" data-action="add-local">Add this machine as a target</button>
        </div>
      `,m=h.length?h.map(i=>pe(i,$)).join(""):'<p class="muted">No SSH targets yet.</p>';a.innerHTML=`
      <section class="section">${o}</section>
      <section class="section">
        <h2>Servers over SSH</h2>
        <div class="card-grid">${m}</div>
        ${Je()}
      </section>
    `}async function u(d,$){if(d==="add-local"){await S();return}if(d==="delete-target"){const x=$.dataset.id;if(!x||!confirm(`Remove target "${x}"? This does not touch anything already running on it.`))return;await b(x);return}d==="add-ssh"&&await H()}async function S(){B();try{await oe({id:_e,mode:"local"}),await e()}catch(d){k(d)}}async function b(d){try{await $e(d),await e()}catch($){k($)}}async function H(){const d=t.querySelector("#ssh-host"),$=t.querySelector("#ssh-user"),x=t.querySelector("#ssh-key"),h=t.querySelector("#ssh-port"),o=t.querySelector("#ssh-id");if(!d||!$||!x||!h||!o)return;const m=d.value.trim(),i=$.value.trim(),L=x.value.trim(),I=h.value.trim(),q=o.value.trim();if(B(),!m||!i||!L){k(new Error("host, user, and key path are required"));return}const W=q||Ge(m),l={Host:m,User:i,KeyPath:L};if(I){const p=Number.parseInt(I,10);if(!Number.isFinite(p)||p<=0){k(new Error("port must be a positive number"));return}l.Port=p}const v=t.querySelector("#ssh-submit");v&&(v.disabled=!0,v.textContent="Connecting…");try{await oe({id:W,mode:"ssh",ssh:l}),await e()}catch(p){k(p),v&&(v.disabled=!1,v.textContent="Add server")}}function k(d){let $=t.querySelector("#targets-error");$||(a.insertAdjacentHTML("afterbegin",'<p id="targets-error" class="error"></p>'),$=t.querySelector("#targets-error")),$.textContent=String(d instanceof Error?d.message:d)}function B(){var d;(d=t.querySelector("#targets-error"))==null||d.remove()}return()=>{n=!0}}function pe(t,n){const a=t.wire,e=t.mode==="local"?"this machine":"SSH",c=t.mode==="ssh"&&t.ssh?`${s(t.ssh.User)}@${s(t.ssh.Host)}`:e;let u,S;if(!a)u=R("not set up","neutral"),S=`<a class="btn" href="#/setup/${encodeURIComponent(t.id)}">Run setup wizard</a>`;else{const b=n.networks.find(k=>k.ChainID===a.ChainID),H=b?b.Name:`chain ${a.ChainID}`;u=`${R(H,"ok")} ${R(a.ExecID,"neutral")} ${R(a.BeaconID,"neutral")}${a.Archive?" "+R("archive","warn"):""}`,S=`
      <a class="btn" href="#/dash/${encodeURIComponent(t.id)}">Dashboard</a>
      <a class="btn" href="#/logs/${encodeURIComponent(t.id)}">Logs</a>
      <a class="btn btn-ghost" href="#/setup/${encodeURIComponent(t.id)}">Re-run setup</a>
    `}return`
    <div class="card">
      <h2>${s(t.id)}</h2>
      <p class="muted">${c}</p>
      <p>${u}</p>
      <div class="card-actions">
        ${S}
        <button class="btn btn-danger" data-action="delete-target" data-id="${s(t.id)}">Remove</button>
      </div>
    </div>
  `}function Je(){return`
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
  `}function We(){const t=navigator.userAgentData,n=(t==null?void 0:t.platform)||navigator.platform||navigator.userAgent;return/mac|win/i.test(n)&&!/linux|android/i.test(n)}function Ke(){return We()?`
      <p class="banner banner-warn">
        macOS and Windows are not supported node hosts — use this machine as a controller and add a
        Linux server over SSH.
      </p>
    `:'<p class="muted">The machine running valve-node. Setup only works on a Linux target.</p>'}function Ge(t){return t.toLowerCase().replace(/[^a-z0-9]+/g,"-").replace(/^-+|-+$/g,"")||"target"}const ae=[{id:"preflight",title:"Preflight checks"},{id:"toolchain",title:"Ensure git + build toolchains"},{id:"install-exec",title:"Install execution client"},{id:"install-beacon",title:"Install beacon client"},{id:"wire",title:"Write JWT secret and systemd units"},{id:"start",title:"Start execution and beacon services"},{id:"handshake",title:"Verify execution/beacon handshake"}],Z=8545,ee=5052,te=30303,Ve=[369,943,1],fe={369:"default",943:"practise here first"};function Xe(t,n){let a=!1;const e={targetId:n,step:"network",catalog:null,loadError:null,chainId:369,execId:null,beaconId:null,archive:!1,dataDir:"",jwtPath:"",execHTTPPort:"",beaconHTTPPort:"",execP2PPort:"",starting:!1,startError:null,events:[],streamStop:null};t.innerHTML=`<h1>Setup: ${s(n)}</h1><div id="wizard-body"><p class="muted">Loading catalog…</p></div><div id="wizard-footer">${N()}</div>`;const c=t.querySelector("#wizard-body"),u=t.querySelector("#wizard-footer");Y(t,(l,v)=>{m(l,v)}),S();async function S(){try{const[l,v]=await Promise.all([G(),V()]);if(a)return;e.catalog=l;const p=v.find(T=>T.id===n);p!=null&&p.wire&&(e.chainId=p.wire.ChainID,e.execId=p.wire.ExecID,e.beaconId=p.wire.BeaconID,e.archive=p.wire.Archive,p.wire.ExecHTTPPort&&(e.execHTTPPort=String(p.wire.ExecHTTPPort)),p.wire.BeaconHTTPPort&&(e.beaconHTTPPort=String(p.wire.BeaconHTTPPort)),p.wire.ExecP2PPort&&(e.execP2PPort=String(p.wire.ExecP2PPort))),b()}catch(l){if(a)return;e.loadError=String(l instanceof Error?l.message:l),b()}}function b(){if(e.loadError){c.innerHTML=`<p class="error">Failed to load: ${s(e.loadError)}</p>`;return}e.catalog&&(c.innerHTML=`
      ${W(e.step)}
      ${k()}
    `,H())}function H(){var v;const l=(v=e.catalog)==null?void 0:v.networks.find(p=>p.ChainID===e.chainId);u.innerHTML=l?N(l.Name,l.LearnURL):N()}function k(){switch(e.step){case"network":return B();case"clients":return d();case"mode":return x();case"review":return h();case"run":return o()}}function B(){const l=e.catalog;return`
      <section>
        <h2>1. Choose a network</h2>
        <div class="card-grid">${Ve.map(p=>{const T=l.networks.find(A=>A.ChainID===p);if(!T)return"";const E=e.chainId===p,P=fe[p]?R(fe[p],p===369?"ok":"warn"):"";return`
        <button class="card card-selectable ${E?"selected":""}" data-action="pick-network" data-chain-id="${p}" type="button">
          <h3>${s(T.Name)} <span class="muted">(chain ${p})</span></h3>
          ${P}
          <p class="muted small">Checkpoint sync from ${s(T.CheckpointURL)}</p>
        </button>
      `}).join("")}</div>
        <div class="wizard-actions">
          <button class="btn" data-action="goto-clients" ${e.chainId===null?"disabled":""}>Next: clients</button>
        </div>
      </section>
    `}function d(){const l=e.catalog,v=l.networks.find(E=>E.ChainID===e.chainId);if(!v)return'<p class="error">Unknown network.</p>';(e.execId===null||!v.ExecClients.includes(e.execId))&&(e.execId=v.ExecClients[0]??null),(e.beaconId===null||!v.BeaconClients.includes(e.beaconId))&&(e.beaconId=v.BeaconClients[0]??null);const p=v.ExecClients.map(E=>$(E,l,e.execId)).join(""),T=v.BeaconClients.map(E=>$(E,l,e.beaconId)).join("");return`
      <section>
        <h2>2. Choose your client pair</h2>
        <p class="muted">Only combinations known to work on ${s(v.Name)} are offered.</p>
        <label>
          Execution client
          <select id="exec-select" data-action="pick-exec">${p}</select>
        </label>
        <label>
          Beacon client
          <select id="beacon-select" data-action="pick-beacon">${T}</select>
        </label>
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-network">Back</button>
          <button class="btn" data-action="goto-mode">Next: mode</button>
        </div>
      </section>
    `}function $(l,v,p){const T=v.clients.find(P=>P.id===l),E=T?`${T.id} (${T.toolchain})`:l;return`<option value="${s(l)}" ${l===p?"selected":""}>${s(E)}</option>`}function x(){const l=e.chainId!==null?`/var/lib/valve-node/${e.chainId}`:"";return`
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
            Data directory <span class="muted">(default: ${s(l)})</span>
            <input id="data-dir-input" type="text" placeholder="${s(l)}" value="${s(e.dataDir)}" />
          </label>
          <label>
            JWT secret path <span class="muted">(default: &lt;data dir&gt;/jwt.hex)</span>
            <input id="jwt-path-input" type="text" placeholder="${s(l)}/jwt.hex" value="${s(e.jwtPath)}" />
          </label>
          <label>
            Execution HTTP port <span class="muted">(default: ${Z})</span>
            <input id="exec-http-port-input" type="text" inputmode="numeric" placeholder="${Z}" value="${s(e.execHTTPPort)}" />
          </label>
          <label>
            Beacon HTTP port <span class="muted">(default: ${ee})</span>
            <input id="beacon-http-port-input" type="text" inputmode="numeric" placeholder="${ee}" value="${s(e.beaconHTTPPort)}" />
          </label>
          <label>
            Execution p2p port <span class="muted">(default: ${te})</span>
            <input id="exec-p2p-port-input" type="text" inputmode="numeric" placeholder="${te}" value="${s(e.execP2PPort)}" />
          </label>
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
    `}function h(){const v=e.catalog.networks.find(w=>w.ChainID===e.chainId),p=e.dataDir||`/var/lib/valve-node/${e.chainId}`,T=e.jwtPath||`${p}/jwt.hex`,E=ae.map(w=>`<li>${s(w.title)}</li>`).join(""),P=I(e.execHTTPPort,Z),A=I(e.beaconHTTPPort,ee),U=I(e.execP2PPort,te),K=P||A||U?`<tr><th>Non-default ports</th><td>${[P?`exec HTTP ${P}`:null,A?`beacon HTTP ${A}`:null,U?`exec p2p ${U}`:null].filter(w=>w!==null).map(s).join(", ")}</td></tr>`:"";return`
      <section>
        <h2>4. Review</h2>
        <table class="review-table">
          <tbody>
            <tr><th>Target</th><td>${s(e.targetId)}</td></tr>
            <tr><th>Network</th><td>${s((v==null?void 0:v.Name)??String(e.chainId))} (chain ${e.chainId})</td></tr>
            <tr><th>Execution client</th><td>${s(e.execId??"")}</td></tr>
            <tr><th>Beacon client</th><td>${s(e.beaconId??"")}</td></tr>
            <tr><th>Mode</th><td>${e.archive?"Archive":"Full"}</td></tr>
            <tr><th>Data directory</th><td><code>${s(p)}</code></td></tr>
            <tr><th>JWT secret path</th><td><code>${s(T)}</code></td></tr>
            ${K}
          </tbody>
        </table>
        <p class="muted small">
          There is no preview API for the exact files/units that will be
          written — the list below is the fixed step sequence setup always
          runs; the actual commands and file contents stream live once you
          start.
        </p>
        <ol class="step-preview">${E}</ol>
        ${e.startError?`<p class="error">${s(e.startError)}</p>`:""}
        <div class="wizard-actions">
          <button class="btn btn-ghost" data-action="goto-mode">Back</button>
          <button class="btn btn-primary" data-action="start-setup" ${e.starting?"disabled":""}>
            ${e.starting?"Starting…":"Start setup"}
          </button>
        </div>
      </section>
    `}function o(){const v=e.catalog.networks.find(w=>w.ChainID===e.chainId),p=v==null?void 0:v.LearnURL,T=new Set(e.events.filter(w=>w.done).map(w=>w.stepId)),E=new Set(e.events.filter(w=>w.err).map(w=>w.stepId)),P=new Map;for(const w of e.events){if(!w.line)continue;const O=P.get(w.stepId)??[];O.push(w.line),P.set(w.stepId,O)}const A=ae.map(w=>{var g;const O=T.has(w.id),Q=E.has(w.id),_=Q?R("failed","bad"):O?R("done","ok"):R("pending","neutral"),r=(P.get(w.id)??[]).slice(-5),f=(g=e.events.find(C=>C.stepId===w.id&&C.err))==null?void 0:g.err,y=w.id==="handshake"?`<p class="muted small">"Talking" means the beacon client can reach the execution client's Engine API over the shared JWT secret and both report the same head — the sign your node is wired correctly.${p?` <a href="${s(p)}" target="_blank" rel="noopener noreferrer">Learn more →</a>`:""}</p>`:"";return`
        <li class="step-row ${O?"step-done":""} ${Q?"step-error":""}">
          <div class="step-head">${_} <strong>${s(w.title)}</strong></div>
          ${y}
          ${r.length?`<pre class="step-log">${r.map(C=>s(C)).join(`
`)}</pre>`:""}
          ${f?`<p class="error small">${s(f)}</p>`:""}
        </li>
      `}).join(""),U=e.events.some(w=>w.err),K=ae.every(w=>T.has(w.id))||e.events.some(w=>w.stepId==="handshake"&&w.done);return`
      <section>
        <h2>5. Running setup</h2>
        <ol class="step-list">${A}</ol>
        ${K&&!U?`<p class="ok">Setup complete. <a href="#/dash/${encodeURIComponent(e.targetId)}">Open the dashboard →</a></p>`:""}
        ${U?'<button class="btn" data-action="start-setup">Retry setup</button>':""}
      </section>
    `}function m(l,v){switch(l){case"pick-network":e.chainId=Number(v.dataset.chainId),e.execId=null,e.beaconId=null,b();break;case"goto-network":e.step="network",b();break;case"goto-clients":if(e.chainId===null)return;e.step="clients",b();break;case"goto-mode":i(),e.step="mode",b();break;case"goto-review":L(),e.step="review",b();break;case"start-setup":q();break}}function i(){const l=t.querySelector("#exec-select"),v=t.querySelector("#beacon-select");l&&(e.execId=l.value),v&&(e.beaconId=v.value)}function L(){const l=t.querySelectorAll('input[name="mode"]');for(const A of Array.from(l))A.checked&&(e.archive=A.value==="archive");const v=t.querySelector("#data-dir-input"),p=t.querySelector("#jwt-path-input");v&&(e.dataDir=v.value.trim()),p&&(e.jwtPath=p.value.trim());const T=t.querySelector("#exec-http-port-input"),E=t.querySelector("#beacon-http-port-input"),P=t.querySelector("#exec-p2p-port-input");T&&(e.execHTTPPort=T.value.trim()),E&&(e.beaconHTTPPort=E.value.trim()),P&&(e.execP2PPort=P.value.trim())}function I(l,v){if(!l)return;const p=Number.parseInt(l,10);if(!(!Number.isFinite(p)||p<=0||p===v))return p}async function q(){var E;if(e.chainId===null||!e.execId||!e.beaconId)return;e.starting=!0,e.startError=null,b();const l={ChainID:e.chainId,ExecID:e.execId,BeaconID:e.beaconId,Archive:e.archive};e.dataDir&&(l.DataDir=e.dataDir),e.jwtPath&&(l.JWTPath=e.jwtPath);const v=I(e.execHTTPPort,Z),p=I(e.beaconHTTPPort,ee),T=I(e.execP2PPort,te);v!==void 0&&(l.ExecHTTPPort=v),p!==void 0&&(l.BeaconHTTPPort=p),T!==void 0&&(l.ExecP2PPort=T);try{await we(e.targetId,l)}catch(P){if(!(P instanceof re&&P.status===409)){e.starting=!1,e.startError=String(P instanceof Error?P.message:P),b();return}}e.starting=!1,e.step="run",e.events=[],b(),(E=e.streamStop)==null||E.call(e),e.streamStop=xe(e.targetId,P=>{a||(e.events.push(P),e.step==="run"&&b())})}function W(l){const v=[{id:"network",label:"Network"},{id:"clients",label:"Clients"},{id:"mode",label:"Mode"},{id:"review",label:"Review"},{id:"run",label:"Run"}],T=v.map(E=>E.id).indexOf(l);return`
      <ol class="wizard-progress">
        ${v.map((E,P)=>`<li class="${P===T?"current":P<T?"past":"future"}">${s(E.label)}</li>`).join("")}
      </ol>
    `}return()=>{var l;a=!0,(l=e.streamStop)==null||l.call(e)}}const Ye=document.querySelector("#app"),{contentEl:Qe,setActiveNav:Ze}=De(Ye);let M=null;function et(){const n=location.hash.replace(/^#\/?/,"").split("/").filter(Boolean);if(n.length===0)return{screen:"targets"};const[a,e]=n;return a==="setup"||a==="dash"||a==="logs"||a==="security"?{screen:a,id:e?decodeURIComponent(e):void 0}:{screen:a??"targets"}}function J(t){const n=document.createElement("div");return Qe.replaceChildren(n),t(n)}function me(){if(M){try{M()}catch{}M=null}const{screen:t,id:n}=et();switch(Ze(t),t){case"setup":if(!n){location.hash="#/targets";return}M=J(a=>Xe(a,n));break;case"dash":if(!n){location.hash="#/targets";return}M=J(a=>Ue(a,n));break;case"logs":if(!n){location.hash="#/targets";return}M=J(a=>qe(a,n));break;case"security":if(!n){location.hash="#/targets";return}M=J(a=>Oe(a,n));break;case"settings":M=J(a=>Fe(a));break;case"targets":default:M=J(a=>ze(a));break}}window.addEventListener("hashchange",me);me();
