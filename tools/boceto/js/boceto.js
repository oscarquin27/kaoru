/* Kaoru — boceto. Sin dependencias.
   1) Preferencias (tema, transparencia, densidad)  2) Armazón común (una sola lista de secciones)
   3) Pestañas  4) Paleta Ctrl+K  5) Panel lateral */
(function () {
  'use strict';

  /* ---------- 1. Preferencias: se aplican antes de pintar ---------- */
  var root = document.documentElement;
  var PREFS = { theme: 'dark', glass: 'on', density: 'comfortable' };

  function readPref(key) {
    try { return localStorage.getItem('kaoru.' + key) || PREFS[key]; } catch (e) { return PREFS[key]; }
  }
  function writePref(key, value) {
    root.dataset[key] = value;
    try { localStorage.setItem('kaoru.' + key, value); } catch (e) { /* sin almacenamiento: vale solo en esta página */ }
  }
  Object.keys(PREFS).forEach(function (key) { root.dataset[key] = readPref(key); });

  /* ---------- 2. Armazón ---------- */
  /* Única lista de secciones (equivale a web/src/app/sections.tsx). href vacío = sin pantalla en el boceto. */
  var SECTIONS = [
    { id: 'inicio', label: 'Inicio', icon: 'house', href: 'index.html' },
    { id: 'cambios', label: 'Cambios', icon: 'swap', href: '' },
    { id: 'requerimientos', label: 'Requerimientos', icon: 'file', href: 'requerimiento.html' },
    { id: 'pruebas', label: 'Pruebas', icon: 'flask', href: 'matriz.html' },
    { id: 'defectos', label: 'Defectos', icon: 'bug', href: '' },
    { id: 'paquetes', label: 'Paquetes', icon: 'box', href: '' },
    { id: 'ambientes', label: 'Ambientes', icon: 'server', href: '' },
    { id: 'produccion', label: 'Producción', icon: 'flag', href: '' },
    { id: 'informes', label: 'Informes', icon: 'chart', href: '' },
    { id: 'calendario', label: 'Calendario', icon: 'calendar', href: '' },
    { id: 'administracion', label: 'Administración', icon: 'sliders', href: '' }
  ];

  var PALETTE = [
    { group: 'Ir a', label: 'REQ-2026-0042', note: 'Límite diario en pagos móviles', icon: 'file', href: 'requerimiento.html', id: true },
    { group: 'Ir a', label: 'DEF-2026-0091', note: 'Defecto · severidad alta', icon: 'bug', href: 'requerimiento.html#defectos', id: true },
    { group: 'Ir a', label: 'TC-2026-0110', note: 'Caso de prueba', icon: 'flask', href: 'requerimiento.html#pruebas', id: true },
    { group: 'Ir a', label: 'RUN-2026-0533', note: 'Ejecución oficial', icon: 'flask', href: 'requerimiento.html#pruebas', id: true },
    { group: 'Ir a', label: 'DOC-2026-0031', note: 'Especificación funcional', icon: 'file', href: 'documento.html', id: true },
    { group: 'Acciones', label: 'Nuevo DEF', note: 'Sin pantalla en el boceto', icon: 'plus', href: '' },
    { group: 'Acciones', label: 'Ir a matriz', note: 'Matriz de trazabilidad', icon: 'grid', href: 'matriz.html' },
    { group: 'Acciones', label: 'Ir a inicio', note: 'Tablero', icon: 'house', href: 'index.html' }
  ];

  /* Iconos: un solo trazo y tamaño; el nombre dice la forma, no el uso. */
  var ICONS = {
    house: '<path d="M4 11l8-7 8 7"/><path d="M6 10v10h12V10"/>',
    swap: '<path d="M4 8h14l-3-3"/><path d="M20 16H6l3 3"/>',
    file: '<path d="M7 3h7l4 4v14H7z"/><path d="M14 3v4h4"/>',
    flask: '<path d="M9 3h6"/><path d="M10 3v6L5 19a1.5 1.5 0 0 0 1.4 2h11.2A1.5 1.5 0 0 0 19 19l-5-10V3"/>',
    bug: '<path d="M8 10a4 4 0 0 1 8 0v5a4 4 0 0 1-8 0z"/><path d="M4 13h4M16 13h4M5 7l3 2M19 7l-3 2M5 19l3-2M19 19l-3-2"/>',
    box: '<path d="M4 8l8-4 8 4v8l-8 4-8-4z"/><path d="M4 8l8 4 8-4M12 12v8"/>',
    server: '<path d="M4 5h16v6H4zM4 13h16v6H4z"/><path d="M8 8h.01M8 16h.01"/>',
    flag: '<path d="M6 21V4"/><path d="M6 5h11l-2 4 2 4H6"/>',
    chart: '<path d="M4 20h16M7 16v-5M12 16V6M17 16V9"/>',
    calendar: '<path d="M5 6h14v14H5zM5 10h14M9 3v4M15 3v4"/>',
    sliders: '<path d="M4 7h10M18 7h2M4 17h2M10 17h10"/><circle cx="16" cy="7" r="2"/><circle cx="8" cy="17" r="2"/>',
    search: '<circle cx="11" cy="11" r="6"/><path d="M16 16l4 4"/>',
    sun: '<circle cx="12" cy="12" r="4"/><path d="M12 3v2M12 19v2M3 12h2M19 12h2M5.6 5.6l1.4 1.4M17 17l1.4 1.4M5.6 18.4L7 17M17 7l1.4-1.4"/>',
    moon: '<path d="M20 14A8 8 0 1 1 10 4a6 6 0 0 0 10 10z"/>',
    check: '<path d="M5 12l5 5 9-10"/>',
    clock: '<circle cx="12" cy="12" r="8"/><path d="M12 8v4l3 2"/>',
    lock: '<path d="M7 11h10v9H7z"/><path d="M9 11V8a3 3 0 0 1 6 0v3"/>',
    triangle: '<path d="M12 4l9 16H3z"/><path d="M12 10v4M12 17h.01"/>',
    circle: '<circle cx="12" cy="12" r="8"/>',
    target: '<circle cx="12" cy="12" r="8"/><circle cx="12" cy="12" r="3"/>',
    link: '<path d="M10 14a4 4 0 0 0 5.7 0l3-3A4 4 0 0 0 13 5.3l-1 1"/><path d="M14 10a4 4 0 0 0-5.7 0l-3 3A4 4 0 0 0 11 18.7l1-1"/>',
    grid: '<path d="M4 5h16v14H4zM4 10h16M4 15h16M10 5v14"/>',
    'arrow-out': '<path d="M14 5h5v5M19 5l-8 8"/><path d="M11 7H5v12h12v-6"/>',
    'arrow-right': '<path d="M5 12h14M13 6l6 6-6 6"/>',
    x: '<path d="M6 6l12 12M18 6L6 18"/>',
    download: '<path d="M12 4v11M7 11l5 5 5-5M5 20h14"/>',
    rows: '<path d="M4 6h16M4 12h16M4 18h16"/>',
    layers: '<path d="M12 4l9 5-9 5-9-5z"/><path d="M3 14l9 5 9-5"/>',
    info: '<circle cx="12" cy="12" r="8"/><path d="M12 11v5M12 8h.01"/>',
    minus: '<path d="M6 12h12"/>',
    plus: '<path d="M12 5v14M5 12h14"/>',
    /* Símbolo de marca: K de una línea continua que une cuatro puntos (REQ → TC → RUN → PKG). */
    kaoru: '<path d="M7 4v16V12l10-8L7 12l10 8"/><circle cx="7" cy="4" r="1.6" fill="currentColor"/><circle cx="7" cy="20" r="1.6" fill="currentColor"/><circle cx="17" cy="4" r="1.6" fill="currentColor"/><circle cx="17" cy="20" r="1.6" fill="currentColor"/>'
  };

  function icon(name, cls) {
    return '<svg class="' + (cls || 'icon') + '" viewBox="0 0 24 24" aria-hidden="true"><use href="#i-' + name + '"/></svg>';
  }

  function sprite() {
    var html = '<svg xmlns="http://www.w3.org/2000/svg" width="0" height="0" style="position:absolute" aria-hidden="true">';
    Object.keys(ICONS).forEach(function (name) {
      html += '<symbol id="i-' + name + '" viewBox="0 0 24 24">' + ICONS[name] + '</symbol>';
    });
    return html + '</svg>';
  }

  function navItem(s, current) {
    var attrs = s.href
      ? 'href="' + s.href + '"'
      : 'href="#" aria-disabled="true" title="Sin pantalla en el boceto"';
    if (s.id === current) attrs += ' aria-current="page"';
    return '<li><a class="nav__item" ' + attrs + '>' + icon(s.icon) + '<span>' + s.label + '</span></a></li>';
  }

  function buildShell() {
    var current = document.body.dataset.section || '';
    document.body.insertAdjacentHTML('afterbegin', sprite());

    var sidebar = document.querySelector('[data-armazon="sidebar"]');
    if (sidebar) {
      sidebar.innerHTML =
        '<a class="brand" href="index.html" aria-label="Kaoru, inicio">' + icon('kaoru', 'icon brand__mark') +
        '<span class="wordmark">kaoru</span></a>' +
        '<nav aria-label="Áreas"><ul class="nav">' +
        SECTIONS.map(function (s) { return navItem(s, current); }).join('') + '</ul></nav>' +
        '<ul class="sidebar__foot nav">' +
        navItem({ id: 'acerca', label: 'Acerca de', icon: 'info', href: 'acerca.html' }, current) + '</ul>';
    }

    var tools = document.querySelector('[data-armazon="tools"]');
    if (tools) {
      tools.innerHTML =
        '<button class="btn search-btn" type="button" data-open-palette>' + icon('search') +
        '<span>Buscar ID o texto</span><span class="kbd">Ctrl K</span></button>' +
        '<button class="btn btn--icon" type="button" data-toggle="density" title="Densidad compacta" aria-label="Densidad compacta" aria-pressed="false">' + icon('rows') + '</button>' +
        '<button class="btn btn--icon" type="button" data-toggle="glass" title="Reducir transparencia" aria-label="Reducir transparencia" aria-pressed="false">' + icon('layers') + '</button>' +
        '<button class="btn btn--icon" type="button" data-toggle="theme" title="Tema claro" aria-label="Tema claro" aria-pressed="false">' + icon('sun') + '</button>';
    }

    var foot = document.querySelector('[data-armazon="foot"]');
    if (foot) {
      foot.innerHTML = '<span>Kaoru · Trazabilidad a prueba de errores</span>' +
        '<span>Boceto. Datos de ejemplo: no corresponden a requerimientos reales.</span>';
    }

    document.body.insertAdjacentHTML('beforeend',
      '<dialog class="palette glass" id="palette" aria-label="Búsqueda global">' +
      '<input class="palette__input" type="text" role="combobox" aria-expanded="true" aria-controls="palette-list" ' +
      'aria-label="Buscar ID o texto" placeholder="Escribe un ID o un texto" autocomplete="off" spellcheck="false">' +
      '<div class="palette__list" id="palette-list" role="listbox"></div>' +
      '<div class="palette__foot"><span><span class="kbd">↑↓</span> mover</span><span><span class="kbd">Enter</span> abrir</span><span><span class="kbd">Esc</span> cerrar</span></div>' +
      '</dialog>');
  }

  /* Interruptores de la barra superior */
  var TOGGLES = {
    theme: { on: 'light', off: 'dark' },
    glass: { on: 'off', off: 'on' },
    density: { on: 'compact', off: 'comfortable' }
  };
  function syncToggles() {
    document.querySelectorAll('[data-toggle]').forEach(function (btn) {
      var t = TOGGLES[btn.dataset.toggle];
      btn.setAttribute('aria-pressed', String(root.dataset[btn.dataset.toggle] === t.on));
    });
  }
  function initToggles() {
    document.querySelectorAll('[data-toggle]').forEach(function (btn) {
      btn.addEventListener('click', function () {
        var key = btn.dataset.toggle, t = TOGGLES[key];
        writePref(key, root.dataset[key] === t.on ? t.off : t.on);
        syncToggles();
      });
    });
    syncToggles();
  }

  /* ---------- 3. Pestañas ---------- */
  function initTabs() {
    document.querySelectorAll('[role="tablist"]').forEach(function (list) {
      var tabs = Array.prototype.slice.call(list.querySelectorAll('[role="tab"]'));
      function select(tab, focus) {
        tabs.forEach(function (t) {
          var on = t === tab;
          t.setAttribute('aria-selected', String(on));
          t.tabIndex = on ? 0 : -1;
          document.getElementById(t.getAttribute('aria-controls')).hidden = !on;
        });
        if (focus) tab.focus();
      }
      function fromHash() {
        var wanted = tabs.filter(function (t) { return '#' + t.dataset.tab === location.hash; })[0];
        if (wanted) select(wanted, false);
      }
      tabs.forEach(function (tab, i) {
        tab.addEventListener('click', function () {
          select(tab, false);
          history.replaceState(null, '', '#' + tab.dataset.tab);
        });
        tab.addEventListener('keydown', function (e) {
          var step = e.key === 'ArrowRight' ? 1 : e.key === 'ArrowLeft' ? -1 : 0;
          if (!step) return;
          e.preventDefault();
          select(tabs[(i + step + tabs.length) % tabs.length], true);
        });
      });
      window.addEventListener('hashchange', fromHash);
      fromHash();
    });
  }

  /* ---------- 4. Paleta Ctrl+K (simulada: lista fija) ---------- */
  function initPalette() {
    var dialog = document.getElementById('palette');
    var input = dialog.querySelector('input');
    var list = dialog.querySelector('.palette__list');
    var opener = null, active = 0, shown = [];

    function render() {
      var q = input.value.trim().toLowerCase();
      shown = PALETTE.filter(function (p) {
        return !q || (p.label + ' ' + p.note).toLowerCase().indexOf(q) !== -1;
      });
      active = Math.min(active, Math.max(shown.length - 1, 0));
      if (!shown.length) {
        list.innerHTML = '<p class="palette__empty">Sin resultados. Prueba con un ID completo, por ejemplo REQ-2026-0042.</p>';
        return;
      }
      var html = '', group = '';
      shown.forEach(function (p, i) {
        if (p.group !== group) { group = p.group; html += '<p class="palette__group h3">' + group + '</p>'; }
        html += '<a class="palette__item" role="option" id="palette-opt-' + i + '" aria-selected="' + (i === active) + '" href="' + (p.href || '#') + '">' +
          icon(p.icon) + '<span class="' + (p.id ? 'id' : '') + '">' + p.label + '</span><span class="muted">' + p.note + '</span></a>';
      });
      list.innerHTML = html;
      input.setAttribute('aria-activedescendant', 'palette-opt-' + active);
      var el = document.getElementById('palette-opt-' + active);
      if (el) el.scrollIntoView({ block: 'nearest' });
    }

    function open() {
      if (dialog.open) return;
      opener = document.activeElement;
      input.value = ''; active = 0; render();
      dialog.showModal();
    }

    document.addEventListener('keydown', function (e) {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') { e.preventDefault(); open(); }
    });
    document.querySelectorAll('[data-open-palette]').forEach(function (b) { b.addEventListener('click', open); });
    input.addEventListener('input', function () { active = 0; render(); });
    input.addEventListener('keydown', function (e) {
      if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
        e.preventDefault();
        if (!shown.length) return;
        active = (active + (e.key === 'ArrowDown' ? 1 : -1) + shown.length) % shown.length;
        render();
      } else if (e.key === 'Enter' && shown[active]) {
        e.preventDefault();
        if (shown[active].href) location.href = shown[active].href; else dialog.close();
      }
    });
    dialog.addEventListener('click', function (e) { if (e.target === dialog) dialog.close(); });
    dialog.addEventListener('close', function () { if (opener && opener.focus) opener.focus(); });
  }

  /* ---------- 5. Panel lateral: abre la ficha sin perder la posición y devuelve el foco ---------- */
  function initDrawer() {
    var drawer = document.getElementById('drawer');
    if (!drawer) return;
    var opener = null;
    var code = drawer.querySelector('[data-drawer-code]');
    var fields = drawer.querySelector('[data-drawer-fields]');
    var full = drawer.querySelector('[data-drawer-full]');

    document.addEventListener('click', function (e) {
      var trigger = e.target.closest('[data-ficha]');
      if (!trigger) return;
      opener = trigger;
      var ficha = JSON.parse(trigger.dataset.ficha);
      code.textContent = trigger.textContent.trim();
      fields.innerHTML = Object.keys(ficha).map(function (k) {
        return '<div class="meta__item"><dt>' + k + '</dt><dd>' + ficha[k] + '</dd></div>';
      }).join('');
      full.hidden = !trigger.dataset.href;
      if (trigger.dataset.href) full.href = trigger.dataset.href;
      drawer.showModal();
    });
    drawer.querySelector('[data-drawer-close]').addEventListener('click', function () { drawer.close(); });
    drawer.addEventListener('click', function (e) { if (e.target === drawer) drawer.close(); });
    drawer.addEventListener('close', function () { if (opener) opener.focus(); });
  }

  document.addEventListener('DOMContentLoaded', function () {
    buildShell();
    initToggles();
    initTabs();
    initPalette();
    initDrawer();
  });
})();
