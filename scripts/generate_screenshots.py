#!/usr/bin/env python3
"""
Generates crisp, responsive SVG terminal screenshots and diagrams for ephemvm documentation.
"""

import os

OUTPUT_DIR = "docs/assets/screenshots"
os.makedirs(OUTPUT_DIR, exist_ok=True)

def wrap_terminal(width, height, title, content_svg, status_bar=None):
    status_bar_svg = ""
    if status_bar:
        status_bar_svg = f"""
    <!-- Terminal Footer Status Bar -->
    <rect x="1" y="{height - 32}" width="{width - 2}" height="31" fill="#161b22" rx="0 0 10 10"/>
    <line x1="1" y1="{height - 32}" x2="{width - 1}" y2="{height - 32}" stroke="#30363d" stroke-width="1"/>
    {status_bar}
        """

    return f"""<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {width} {height}" width="100%" height="auto" style="max-width: {width}px; display: block; margin: 1.5rem auto;">
  <defs>
    <filter id="shadow" x="-5%" y="-5%" width="110%" height="115%" filterUnits="userSpaceOnUse">
      <feDropShadow dx="0" dy="12" stdDeviation="16" flood-color="#000000" flood-opacity="0.45" />
    </filter>
    <linearGradient id="headerGrad" x1="0%" y1="0%" x2="100%" y2="0%">
      <stop offset="0%" stop-color="#161b22"/>
      <stop offset="100%" stop-color="#21262d"/>
    </linearGradient>
  </defs>

  <!-- Window Container -->
  <g filter="url(#shadow)">
    <!-- Main Background -->
    <rect x="1" y="1" width="{width - 2}" height="{height - 2}" rx="10" fill="#0d1117" stroke="#30363d" stroke-width="1.5"/>

    <!-- Title Bar -->
    <path d="M 1,11 A 10,10 0 0,1 11,1 L {width - 11},1 A 10,10 0 0,1 {width - 1},11 L {width - 1},38 L 1,38 Z" fill="url(#headerGrad)"/>
    <line x1="1" y1="38" x2="{width - 1}" y2="38" stroke="#30363d" stroke-width="1"/>

    <!-- Window Controls -->
    <circle cx="18" cy="19" r="6" fill="#ff5f56" stroke="#e0443e" stroke-width="0.5"/>
    <circle cx="38" cy="19" r="6" fill="#ffbd2e" stroke="#dea123" stroke-width="0.5"/>
    <circle cx="58" cy="19" r="6" fill="#27c93f" stroke="#1aab29" stroke-width="0.5"/>

    <!-- Window Title -->
    <text x="{width / 2}" y="23" fill="#8b949e" font-family="-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif" font-size="12" font-weight="600" text-anchor="middle" letter-spacing="0.3">{title}</text>

    <!-- Terminal Content Area -->
    <g transform="translate(24, 60)" font-family="ui-monospace, SFMono-Regular, 'SF Mono', Menlo, Consolas, 'Liberation Mono', monospace" font-size="13" line-height="1.5">
      {content_svg}
    </g>

    {status_bar_svg}
  </g>
</svg>"""

def gen_hero_terminal():
    width = 780
    height = 360
    title = "ephemvm — Fast Ephemeral MicroVM Runner"

    content = """
      <!-- Command Prompt Line -->
      <text y="0">
        <tspan fill="#7ee787" font-weight="bold">user@workstation</tspan><tspan fill="#8b949e">:</tspan><tspan fill="#79c0ff">~</tspan><tspan fill="#8b949e">$ </tspan>
        <tspan fill="#f0883e" font-weight="bold">ephemvm</tspan><tspan fill="#e6edf3"> "uname -srm &amp;&amp; uptime"</tspan>
      </text>

      <!-- EphemVM execution notes -->
      <text y="28" fill="#8b949e">
        <tspan fill="#58a6ff">ℹ</tspan> [ephemvm] allocating CoW reflink overlay in /tmp/ephemvm-8f2c...
      </text>
      <text y="52" fill="#8b949e">
        <tspan fill="#58a6ff">ℹ</tspan> [ephemvm] booting Firecracker microVM with 1 vCPU, 128MB RAM...
      </text>

      <!-- Output from microVM -->
      <text y="88" fill="#e6edf3" font-weight="600">
        Linux 6.1.102-ephem x86_64
      </text>
      <text y="112" fill="#e6edf3">
         02:50:11 up 0 min, 0 users, load average: 0.00, 0.00, 0.00
      </text>

      <!-- Teardown -->
      <text y="148" fill="#8b949e">
        <tspan fill="#3fb950">✓</tspan> guest exited cleanly (exit code: 0)
      </text>
      <text y="172" fill="#8b949e">
        <tspan fill="#3fb950">✓</tspan> microVM evaporated, overlay pruned, 0 bytes leaked
      </text>

      <!-- Next prompt -->
      <text y="212">
        <tspan fill="#7ee787" font-weight="bold">user@workstation</tspan><tspan fill="#8b949e">:</tspan><tspan fill="#79c0ff">~</tspan><tspan fill="#8b949e">$ </tspan>
        <tspan fill="#79c0ff" opacity="0.8">_</tspan>
      </text>
    """

    status = f"""
      <text x="24" y="{height - 12}" fill="#3fb950" font-family="-apple-system, BlinkMacSystemFont, sans-serif" font-size="11" font-weight="600">
        ⚡ BOOT: 38ms
      </text>
      <text x="124" y="{height - 12}" fill="#58a6ff" font-family="-apple-system, BlinkMacSystemFont, sans-serif" font-size="11" font-weight="600">
        ⏱ EXEC: 12ms
      </text>
      <text x="224" y="{height - 12}" fill="#a371f7" font-family="-apple-system, BlinkMacSystemFont, sans-serif" font-size="11" font-weight="600">
        🛡 ISOLATION: Hardware KVM
      </text>
      <text x="{width - 24}" y="{height - 12}" fill="#8b949e" font-family="-apple-system, BlinkMacSystemFont, sans-serif" font-size="11" text-anchor="end">
        Status: Exit 0 (Success)
      </text>
    """

    svg = wrap_terminal(width, height, title, content, status)
    with open(f"{OUTPUT_DIR}/hero-terminal.svg", "w", encoding="utf-8") as f:
        f.write(svg)

def gen_destructive_terminal():
    width = 780
    height = 370
    title = "ephemvm — Destructive Teardown & Root Deletion Test"

    content = """
      <!-- Command Prompt Line -->
      <text y="0">
        <tspan fill="#7ee787" font-weight="bold">user@workstation</tspan><tspan fill="#8b949e">:</tspan><tspan fill="#79c0ff">~</tspan><tspan fill="#8b949e">$ </tspan>
        <tspan fill="#f0883e" font-weight="bold">ephemvm</tspan><tspan fill="#e6edf3"> "rm -rf --no-preserve-root / &amp;&amp; ls -la /"</tspan>
      </text>

      <!-- Guest execution -->
      <text y="28" fill="#f85149">
        rm: cannot remove '/proc/1': Operation not permitted
      </text>
      <text y="50" fill="#f85149">
        rm: cannot remove '/sys/fs/cgroup': Operation not permitted
      </text>
      <text y="72" fill="#d29922">
        ls: /: No such file or directory
      </text>

      <!-- Teardown & Verification -->
      <text y="108" fill="#8b949e">
        <tspan fill="#58a6ff">ℹ</tspan> [ephemvm] guest exited with status 1
      </text>
      <text y="132" fill="#8b949e">
        <tspan fill="#3fb950">✓</tspan> [ephemvm] overlay /tmp/ephemvm-9a3d discarded completely
      </text>
      <text y="156" fill="#8b949e">
        <tspan fill="#3fb950">✓</tspan> [ephemvm] base rootfs ~/.cache/ephemvm/rootfs.ext4 SHA256 verified untouched
      </text>

      <!-- Second verify command -->
      <text y="196">
        <tspan fill="#7ee787" font-weight="bold">user@workstation</tspan><tspan fill="#8b949e">:</tspan><tspan fill="#79c0ff">~</tspan><tspan fill="#8b949e">$ </tspan>
        <tspan fill="#f0883e" font-weight="bold">ephemvm</tspan><tspan fill="#e6edf3"> "ls /"</tspan>
      </text>
      <text y="224" fill="#79c0ff">
        bin  boot  dev  etc  home  lib  media  mnt  opt  proc  root  run  sbin  srv  sys  tmp  usr  var
      </text>
    """

    status = f"""
      <text x="24" y="{height - 12}" fill="#f85149" font-family="-apple-system, BlinkMacSystemFont, sans-serif" font-size="11" font-weight="600">
        💥 GUEST DISK: Annihilated
      </text>
      <text x="200" y="{height - 12}" fill="#3fb950" font-family="-apple-system, BlinkMacSystemFont, sans-serif" font-size="11" font-weight="600">
        🛡 HOST DISK: 100% Intact
      </text>
      <text x="{width - 24}" y="{height - 12}" fill="#8b949e" font-family="-apple-system, BlinkMacSystemFont, sans-serif" font-size="11" text-anchor="end">
        CoW Reflink Rollback
      </text>
    """

    svg = wrap_terminal(width, height, title, content, status)
    with open(f"{OUTPUT_DIR}/destructive-terminal.svg", "w", encoding="utf-8") as f:
        f.write(svg)

def gen_ai_agent_terminal():
    width = 780
    height = 430
    title = "ephemvm — Autonomous AI Agent Sandbox & JSON Telemetry"

    content = """
      <!-- Command Prompt Line -->
      <text y="0">
        <tspan fill="#7ee787" font-weight="bold">agent@sandbox</tspan><tspan fill="#8b949e">:</tspan><tspan fill="#79c0ff">~</tspan><tspan fill="#8b949e">$ </tspan>
        <tspan fill="#f0883e" font-weight="bold">ephemvm</tspan><tspan fill="#e6edf3"> run --json --vcpus 2 --memory 256 -- python3 -c \\</tspan>
      </text>
      <text y="24" fill="#a5d6ff">
        &gt; "import socket, os; print(f'Host: {{socket.gethostname()}}, PID: {{os.getpid()}}')"
      </text>

      <!-- JSON Telemetry Output -->
      <text y="64" fill="#8b949e">{</text>
      <text y="86" fill="#8b949e">  <tspan fill="#79c0ff">"command"</tspan>: <tspan fill="#a5d6ff">"python3 -c \\"import socket...\\""</tspan>,</text>
      <text y="108" fill="#8b949e">  <tspan fill="#79c0ff">"exit_code"</tspan>: <tspan fill="#79c0ff">0</tspan>,</text>
      <text y="130" fill="#8b949e">  <tspan fill="#79c0ff">"duration_ms"</tspan>: <tspan fill="#79c0ff">47</tspan>,</text>
      <text y="152" fill="#8b949e">  <tspan fill="#79c0ff">"timed_out"</tspan>: <tspan fill="#ff7b72">false</tspan>,</text>
      <text y="174" fill="#8b949e">  <tspan fill="#79c0ff">"stdout"</tspan>: <tspan fill="#7ee787">"Host: ephemvm-guest, PID: 1\\n"</tspan>,</text>
      <text y="196" fill="#8b949e">  <tspan fill="#79c0ff">"stderr"</tspan>: <tspan fill="#a5d6ff">""</tspan>,</text>
      <text y="218" fill="#8b949e">  <tspan fill="#79c0ff">"vcpus"</tspan>: <tspan fill="#79c0ff">2</tspan>,</text>
      <text y="240" fill="#8b949e">  <tspan fill="#79c0ff">"memory_mb"</tspan>: <tspan fill="#79c0ff">256</tspan></text>
      <text y="262" fill="#8b949e">}</text>

      <text y="296">
        <tspan fill="#7ee787" font-weight="bold">agent@sandbox</tspan><tspan fill="#8b949e">:</tspan><tspan fill="#79c0ff">~</tspan><tspan fill="#8b949e">$ </tspan>
        <tspan fill="#79c0ff" opacity="0.8">_</tspan>
      </text>
    """

    status = f"""
      <text x="24" y="{height - 12}" fill="#58a6ff" font-family="-apple-system, BlinkMacSystemFont, sans-serif" font-size="11" font-weight="600">
        🤖 MCP / AI Agent Tool
      </text>
      <text x="210" y="{height - 12}" fill="#3fb950" font-family="-apple-system, BlinkMacSystemFont, sans-serif" font-size="11" font-weight="600">
        📊 Machine-Readable JSON
      </text>
      <text x="{width - 24}" y="{height - 12}" fill="#8b949e" font-family="-apple-system, BlinkMacSystemFont, sans-serif" font-size="11" text-anchor="end">
        Zero Host State Retention
      </text>
    """

    svg = wrap_terminal(width, height, title, content, status)
    with open(f"{OUTPUT_DIR}/ai-agent-terminal.svg", "w", encoding="utf-8") as f:
        f.write(svg)

def gen_cache_terminal():
    width = 780
    height = 360
    title = "ephemvm cache status — Asset Verification & Offline Readiness"

    content = """
      <!-- Command Prompt Line -->
      <text y="0">
        <tspan fill="#7ee787" font-weight="bold">user@workstation</tspan><tspan fill="#8b949e">:</tspan><tspan fill="#79c0ff">~</tspan><tspan fill="#8b949e">$ </tspan>
        <tspan fill="#f0883e" font-weight="bold">ephemvm</tspan><tspan fill="#e6edf3"> cache status</tspan>
      </text>

      <!-- Table Header -->
      <text y="36" fill="#8b949e" font-weight="bold">
        ASSET        STATUS   SIZE      PERMS  SHA256 CHECKSUM                   VERIFIED
      </text>
      <text y="50" fill="#30363d">
        ────────────────────────────────────────────────────────────────────────────────
      </text>

      <!-- Row 1 -->
      <text y="74">
        <tspan fill="#79c0ff" font-weight="bold">vmlinux     </tspan>
        <tspan fill="#3fb950">CACHED   </tspan>
        <tspan fill="#e6edf3">12.4 MB   </tspan>
        <tspan fill="#8b949e">0444   </tspan>
        <tspan fill="#8b949e">8f1d3c52a0...e42a9b               </tspan>
        <tspan fill="#3fb950">MATCH ✓</tspan>
      </text>

      <!-- Row 2 -->
      <text y="98">
        <tspan fill="#79c0ff" font-weight="bold">rootfs.ext4 </tspan>
        <tspan fill="#3fb950">CACHED   </tspan>
        <tspan fill="#e6edf3">32.0 MB   </tspan>
        <tspan fill="#8b949e">0444   </tspan>
        <tspan fill="#8b949e">3c92a71f05...910bc4               </tspan>
        <tspan fill="#3fb950">MATCH ✓</tspan>
      </text>

      <!-- Row 3 -->
      <text y="122">
        <tspan fill="#79c0ff" font-weight="bold">firecracker </tspan>
        <tspan fill="#3fb950">CACHED   </tspan>
        <tspan fill="#e6edf3">18.1 MB   </tspan>
        <tspan fill="#8b949e">0755   </tspan>
        <tspan fill="#8b949e">b04e198a2e...f721ea               </tspan>
        <tspan fill="#3fb950">MATCH ✓</tspan>
      </text>

      <text y="156" fill="#3fb950" font-weight="bold">
        ✓ All assets verified and ready. Zero internet access required at runtime.
      </text>

      <text y="196">
        <tspan fill="#7ee787" font-weight="bold">user@workstation</tspan><tspan fill="#8b949e">:</tspan><tspan fill="#79c0ff">~</tspan><tspan fill="#8b949e">$ </tspan>
        <tspan fill="#79c0ff" opacity="0.8">_</tspan>
      </text>
    """

    status = f"""
      <text x="24" y="{height - 12}" fill="#3fb950" font-family="-apple-system, BlinkMacSystemFont, sans-serif" font-size="11" font-weight="600">
        📦 CACHE DIR: ~/.cache/ephemvm
      </text>
      <text x="260" y="{height - 12}" fill="#58a6ff" font-family="-apple-system, BlinkMacSystemFont, sans-serif" font-size="11" font-weight="600">
        🔒 INTEGRITY: Cryptographically Verified
      </text>
      <text x="{width - 24}" y="{height - 12}" fill="#8b949e" font-family="-apple-system, BlinkMacSystemFont, sans-serif" font-size="11" text-anchor="end">
        Air-Gap Ready
      </text>
    """

    svg = wrap_terminal(width, height, title, content, status)
    with open(f"{OUTPUT_DIR}/cache-terminal.svg", "w", encoding="utf-8") as f:
        f.write(svg)

def gen_architecture_diagram():
    width = 820
    height = 420

    svg = f"""<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {width} {height}" width="100%" height="auto" style="max-width: {width}px; display: block; margin: 1.5rem auto;">
  <defs>
    <filter id="cardShadow" x="-5%" y="-5%" width="110%" height="115%">
      <feDropShadow dx="0" dy="8" stdDeviation="12" flood-color="#000000" flood-opacity="0.3" />
    </filter>
    <linearGradient id="hostGrad" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" stop-color="#161b22"/>
      <stop offset="100%" stop-color="#0d1117"/>
    </linearGradient>
    <linearGradient id="vmGrad" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" stop-color="#1f2937"/>
      <stop offset="100%" stop-color="#111827"/>
    </linearGradient>
    <marker id="arrow" viewBox="0 0 10 10" refX="5" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse">
      <path d="M 0 0 L 10 5 L 0 10 z" fill="#58a6ff" />
    </marker>
    <marker id="arrowGreen" viewBox="0 0 10 10" refX="5" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse">
      <path d="M 0 0 L 10 5 L 0 10 z" fill="#3fb950" />
    </marker>
  </defs>

  <!-- Container Box -->
  <rect x="2" y="2" width="{width - 4}" height="{height - 4}" rx="14" fill="#0d1117" stroke="#30363d" stroke-width="1.5" />

  <!-- Title & Subtitle -->
  <text x="30" y="38" fill="#e6edf3" font-family="-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif" font-size="18" font-weight="700">ephemvm Isolation &amp; Execution Pipeline</text>
  <text x="30" y="58" fill="#8b949e" font-family="-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif" font-size="12">Hardware KVM microVM, instantaneous reflink CoW clone, serial streaming, and guaranteed evaporation.</text>

  <!-- Box 1: Host User Invocation -->
  <g transform="translate(30, 85)" filter="url(#cardShadow)">
    <rect width="210" height="280" rx="10" fill="#161b22" stroke="#30363d" stroke-width="1.2" />
    <rect width="210" height="34" rx="10 10 0 0" fill="#21262d" />
    <text x="15" y="22" fill="#58a6ff" font-family="-apple-system, BlinkMacSystemFont, sans-serif" font-size="13" font-weight="700">1. Host &amp; CLI</text>
    <text x="15" y="60" fill="#e6edf3" font-family="ui-monospace, monospace" font-size="11" font-weight="600">$ ephemvm run -- &lt;cmd&gt;</text>
    <text x="15" y="90" fill="#8b949e" font-family="-apple-system, BlinkMacSystemFont, sans-serif" font-size="11">• Transparent arg parser</text>
    <text x="15" y="112" fill="#8b949e" font-family="-apple-system, BlinkMacSystemFont, sans-serif" font-size="11">• Script &amp; env injector</text>
    <text x="15" y="134" fill="#8b949e" font-family="-apple-system, BlinkMacSystemFont, sans-serif" font-size="11">• Auto-downloads assets</text>
    <text x="15" y="156" fill="#8b949e" font-family="-apple-system, BlinkMacSystemFont, sans-serif" font-size="11">• Verified SHA256 hashes</text>
    <rect x="15" y="180" width="180" height="75" rx="6" fill="#0d1117" stroke="#30363d" stroke-width="1" />
    <text x="25" y="202" fill="#7ee787" font-family="ui-monospace, monospace" font-size="10">~/.cache/ephemvm/</text>
    <text x="25" y="222" fill="#8b949e" font-family="ui-monospace, monospace" font-size="10">├── vmlinux (0444)</text>
    <text x="25" y="240" fill="#8b949e" font-family="ui-monospace, monospace" font-size="10">└── rootfs.ext4 (0444)</text>
  </g>

  <!-- Arrow 1 to 2 -->
  <path d="M 245 220 L 290 220" stroke="#58a6ff" stroke-width="2" marker-end="url(#arrow)" />

  <!-- Box 2: Ephemeral CoW Storage -->
  <g transform="translate(295, 85)" filter="url(#cardShadow)">
    <rect width="215" height="280" rx="10" fill="#161b22" stroke="#30363d" stroke-width="1.2" />
    <rect width="215" height="34" rx="10 10 0 0" fill="#21262d" />
    <text x="15" y="22" fill="#d29922" font-family="-apple-system, BlinkMacSystemFont, sans-serif" font-size="13" font-weight="700">2. Ephemeral Overlay</text>
    <text x="15" y="60" fill="#e6edf3" font-family="ui-monospace, monospace" font-size="11" font-weight="600">/tmp/ephemvm-&lt;uuid&gt;/</text>
    <text x="15" y="90" fill="#8b949e" font-family="-apple-system, BlinkMacSystemFont, sans-serif" font-size="11">• Instant FICLONE ioctl</text>
    <text x="15" y="112" fill="#8b949e" font-family="-apple-system, BlinkMacSystemFont, sans-serif" font-size="11">• Zero physical blocks copied</text>
    <text x="15" y="134" fill="#8b949e" font-family="-apple-system, BlinkMacSystemFont, sans-serif" font-size="11">• Sparse block fallback</text>
    <text x="15" y="156" fill="#8b949e" font-family="-apple-system, BlinkMacSystemFont, sans-serif" font-size="11">• Dedicated API socket</text>
    <rect x="15" y="180" width="185" height="75" rx="6" fill="#0d1117" stroke="#d29922" stroke-width="1" stroke-dasharray="3,3" />
    <text x="25" y="202" fill="#d29922" font-family="ui-monospace, monospace" font-size="10">Ephemeral Scratchpad</text>
    <text x="25" y="222" fill="#8b949e" font-family="ui-monospace, monospace" font-size="10">overlay.ext4 (CoW)</text>
    <text x="25" y="240" fill="#f85149" font-family="ui-monospace, monospace" font-size="10">💥 DELETED ON EXIT</text>
  </g>

  <!-- Arrow 2 to 3 -->
  <path d="M 515 220 L 560 220" stroke="#58a6ff" stroke-width="2" marker-end="url(#arrow)" />

  <!-- Box 3: Firecracker Hardware MicroVM -->
  <g transform="translate(565, 85)" filter="url(#cardShadow)">
    <rect width="225" height="280" rx="10" fill="#161b22" stroke="#3fb950" stroke-width="1.5" />
    <rect width="225" height="34" rx="10 10 0 0" fill="#238636" />
    <text x="15" y="22" fill="#ffffff" font-family="-apple-system, BlinkMacSystemFont, sans-serif" font-size="13" font-weight="700">3. Firecracker KVM</text>
    <text x="15" y="60" fill="#e6edf3" font-family="ui-monospace, monospace" font-size="11" font-weight="600">Hardware Virtualization</text>
    <text x="15" y="90" fill="#e6edf3" font-family="-apple-system, BlinkMacSystemFont, sans-serif" font-size="11">• Dedicated vCPU &amp; RAM</text>
    <text x="15" y="112" fill="#e6edf3" font-family="-apple-system, BlinkMacSystemFont, sans-serif" font-size="11">• Stripped Linux Kernel</text>
    <text x="15" y="134" fill="#e6edf3" font-family="-apple-system, BlinkMacSystemFont, sans-serif" font-size="11">• Boots in &lt; 50ms</text>
    <text x="15" y="156" fill="#e6edf3" font-family="-apple-system, BlinkMacSystemFont, sans-serif" font-size="11">• Serial UART ttyS0 bridge</text>
    <rect x="15" y="180" width="195" height="75" rx="6" fill="#0d1117" stroke="#3fb950" stroke-width="1" />
    <text x="25" y="202" fill="#3fb950" font-family="ui-monospace, monospace" font-size="10">Sentinel Protocol</text>
    <text x="25" y="222" fill="#e6edf3" font-family="ui-monospace, monospace" font-size="10">__EXIT_CODE:$?__</text>
    <text x="25" y="240" fill="#79c0ff" font-family="ui-monospace, monospace" font-size="10">➔ Real Exit Status</text>
  </g>

  <!-- Return Loop: Serial Streaming to Host -->
  <path d="M 677 370 C 677 400, 135 400, 135 370" fill="none" stroke="#3fb950" stroke-width="2" stroke-dasharray="4,4" marker-end="url(#arrowGreen)" />
  <rect x="330" y="388" width="180" height="22" rx="4" fill="#0d1117" stroke="#3fb950" stroke-width="1"/>
  <text x="420" y="403" fill="#3fb950" font-family="-apple-system, BlinkMacSystemFont, sans-serif" font-size="10" font-weight="600" text-anchor="middle">Unbuffered Real-time Stream</text>
</svg>"""

    with open(f"{OUTPUT_DIR}/architecture-diagram.svg", "w", encoding="utf-8") as f:
        f.write(svg)

def main():
    print("Generating terminal screenshots and diagrams...")
    gen_hero_terminal()
    gen_destructive_terminal()
    gen_ai_agent_terminal()
    gen_cache_terminal()
    gen_architecture_diagram()
    print("All screenshots generated successfully in", OUTPUT_DIR)

if __name__ == "__main__":
    main()
