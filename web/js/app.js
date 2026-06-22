console.log("APP JS v2.0 - Oracle Excel Importer");

// ============================================================
// State
// ============================================================

let excelHeaders = []; // kolom dari excel
let oracleColumns = []; // kolom dari table oracle yang dipilih
let uploadedFileName = ""; // nama file yang sudah di-upload
let currentMode = "create"; // 'create' | 'insert'

// ============================================================
// Load Oracle Tables saat halaman dibuka
// ============================================================

async function loadTables() {
  try {
    const res = await fetch("/api/tables");
    const tables = await res.json();

    const sel = document.getElementById("tableSelect");
    sel.innerHTML = `<option value="">— Pilih Table Oracle —</option>`;

    tables.forEach((t) => {
      sel.innerHTML += `<option value="${t}">${t}</option>`;
    });
  } catch (err) {
    console.error("Load tables error:", err);
  }
}

loadTables();

// ============================================================
// File Upload & Sheet Population
// ============================================================

document
  .getElementById("excelFile")
  .addEventListener("change", async function (e) {
    const file = e.target.files[0];
    if (!file) {
      document.getElementById("sheetSelect").disabled = true;
      document.getElementById("btnPreview").disabled = true;
      return;
    }

    const infoEl = document.getElementById("previewInfo");
    const sheetSelect = document.getElementById("sheetSelect");
    const btnPreview = document.getElementById("btnPreview");

    infoEl.innerHTML = `<span class="loading">⏳ Mengunggah file...</span>`;
    sheetSelect.disabled = true;
    btnPreview.disabled = true;
    sheetSelect.innerHTML = `<option value="">— Loading sheets... —</option>`;

    try {
      const formData = new FormData();
      formData.append("excel", file);

      const res = await fetch("/api/upload", {
        method: "POST",
        body: formData,
      });
      const result = await res.json();

      if (result.error) {
        infoEl.innerHTML = `<span class="error">❌ ${result.error}</span>`;
        return;
      }

      uploadedFileName = result.fileName;

      // Populate sheets
      let options = "";
      (result.sheets || []).forEach((s) => {
        options += `<option value="${escAttr(s)}">${escHtml(s)}</option>`;
      });
      sheetSelect.innerHTML = options;
      sheetSelect.disabled = false;
      btnPreview.disabled = false;

      infoEl.innerHTML = `<span class="badge-success">✔ File terunggah. Silakan pilih sheet dan klik Preview.</span>`;
    } catch (err) {
      infoEl.innerHTML = `<span class="error">❌ ${err.message}</span>`;
      console.error(err);
    }
  });

// ============================================================
// Preview Button
// ============================================================

document.getElementById("btnPreview").addEventListener("click", previewExcel);

async function previewExcel() {
  if (!uploadedFileName) {
    alert("Unggah file Excel terlebih dahulu.");
    return;
  }

  const sheetName = document.getElementById("sheetSelect").value;

  const infoEl = document.getElementById("previewInfo");
  const wrapperEl = document.getElementById("previewWrapper");
  const tableEl = document.getElementById("previewTable");

  infoEl.innerHTML = `<span class="loading">⏳ Loading preview...</span>`;
  wrapperEl.style.display = "none";

  try {
    const res = await fetch("/api/preview", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        fileName: uploadedFileName,
        sheetName: sheetName,
      }),
    });
    const result = await res.json();

    if (result.error) {
      infoEl.innerHTML = `<span class="error">❌ ${result.error}</span>`;
      return;
    }

    excelHeaders = result.headers || [];
    const rows = result.rows || [];

    infoEl.innerHTML = `
            <span class="badge-info">📁 ${uploadedFileName}</span>
            <span class="badge-info">📄 Sheet: ${escHtml(sheetName)}</span>
            <span class="badge-info">📋 ${excelHeaders.length} kolom</span>
            <span class="badge-info">👁 Menampilkan ${rows.length} baris pertama</span>
        `;

    // Render preview table
    tableEl.innerHTML = buildTable(excelHeaders, rows);
    wrapperEl.style.display = "block";

    // Tampilkan Step 2 (Mode)
    document.getElementById("stepMode").style.display = "block";

    // Render kolom checklist untuk mode create
    renderCreateCols();

    // Reset mode ke create
    setMode("create");
  } catch (err) {
    infoEl.innerHTML = `<span class="error">❌ ${err.message}</span>`;
    console.error(err);
  }
}

// ============================================================
// Render preview table
// ============================================================

function buildTable(headers, rows) {
  let html = `<table class="preview-grid"><thead><tr>`;
  html += `<th>#</th>`;
  headers.forEach((h) => {
    html += `<th>${escHtml(h)}</th>`;
  });
  html += `</tr></thead><tbody>`;

  rows.forEach((row, idx) => {
    html += `<tr><td class="row-num">${idx + 1}</td>`;
    headers.forEach((_, ci) => {
      html += `<td>${escHtml(row[ci] ?? "")}</td>`;
    });
    html += `</tr>`;
  });

  html += `</tbody></table>`;
  return html;
}

// ============================================================
// Mode tabs
// ============================================================

function setMode(mode) {
  currentMode = mode;

  // Tab style
  document
    .getElementById("tabCreate")
    .classList.toggle("active", mode === "create");
  document
    .getElementById("tabInsert")
    .classList.toggle("active", mode === "insert");

  // Show/hide panels
  document.getElementById("stepCreate").style.display =
    mode === "create" ? "block" : "none";
  document.getElementById("stepInsert").style.display =
    mode === "insert" ? "block" : "none";
  document.getElementById("stepExecute").style.display = "block";

  // Reset result
  document.getElementById("resultBox").style.display = "none";
  document.getElementById("resultBox").innerHTML = "";
}

// ============================================================
// Step 3 — Create Mode: checkbox per kolom Excel
// ============================================================

function renderCreateCols() {
  const container = document.getElementById("createColList");

  if (excelHeaders.length === 0) {
    container.innerHTML = "<em>Preview Excel terlebih dahulu</em>";
    return;
  }

  let html = "";

  excelHeaders.forEach((h, i) => {
    const id = `chk_col_${i}`;
    html += `
            <label class="col-checkbox-item" for="${id}">
                <input type="checkbox" id="${id}" value="${escAttr(h)}" checked
                    onchange="updateCreateSelectedInfo()">
                <span>${escHtml(h)}</span>
            </label>
        `;
  });

  container.innerHTML = html;
  updateCreateSelectedInfo();
}

function selectAllCols(val) {
  document
    .querySelectorAll("#createColList input[type=checkbox]")
    .forEach((cb) => {
      cb.checked = val;
    });
  updateCreateSelectedInfo();
}

function updateCreateSelectedInfo() {
  const checked = [
    ...document.querySelectorAll("#createColList input[type=checkbox]:checked"),
  ].map((cb) => cb.value);
  const el = document.getElementById("createSelectedInfo");
  el.innerHTML =
    checked.length > 0
      ? `<span class="badge-success">✔ ${checked.length} kolom dipilih</span>`
      : `<span class="badge-warn">⚠ Tidak ada kolom dipilih</span>`;
}

// ============================================================
// Step 3 — Insert Mode: pilih table → load oracle columns → mapping
// ============================================================

document
  .getElementById("tableSelect")
  .addEventListener("change", async function () {
    const table = this.value;

    if (!table) {
      document.getElementById("mappingSection").style.display = "none";
      oracleColumns = [];
      return;
    }

    try {
      const res = await fetch("/api/columns/" + encodeURIComponent(table));
      oracleColumns = await res.json();
      renderMapping();
      document.getElementById("mappingSection").style.display = "block";
    } catch (err) {
      console.error("Load columns error:", err);
    }
  });

function renderMapping() {
  const container = document.getElementById("mappingContainer");

  if (excelHeaders.length === 0 || oracleColumns.length === 0) {
    container.innerHTML =
      "<em>Pastikan sudah preview Excel dan memilih table Oracle.</em>";
    return;
  }

  let html = `
        <div class="mapping-header">
            <span>Kolom Excel</span>
            <span>→ Kolom Oracle (Skip = tidak di-insert)</span>
        </div>
    `;

  excelHeaders.forEach((h, i) => {
    // Coba otomatis match kolom oracle dengan nama mirip
    const autoMatch = findBestMatch(h, oracleColumns);

    html += `
            <div class="mapping-row">
                <div class="excel-col">${escHtml(h)}</div>
                <div class="arrow">→</div>
                <select class="column-map" data-excel="${escAttr(h)}">
                    <option value="">— Skip —</option>
        `;

    oracleColumns.forEach((oc) => {
      const sel = oc === autoMatch ? "selected" : "";
      html += `<option value="${escAttr(oc)}" ${sel}>${escHtml(oc)}</option>`;
    });

    html += `</select></div>`;
  });

  container.innerHTML = html;
}

// Cari oracle column yang paling mirip dengan nama excel column
function findBestMatch(excelCol, oracleCols) {
  const normalized = excelCol.toUpperCase().replace(/\s+/g, "_");
  for (const oc of oracleCols) {
    if (oc.toUpperCase() === normalized) return oc;
  }
  return "";
}

// ============================================================
// Execute Import
// ============================================================

async function executeImport() {
  const resultBox = document.getElementById("resultBox");
  resultBox.innerHTML = `<span class="loading">⏳ Proses import...</span>`;
  resultBox.style.display = "block";
  document.getElementById("btnExecute").disabled = true;

  try {
    if (!uploadedFileName) {
      throw new Error("Preview Excel terlebih dahulu.");
    }

    let payload = {
      fileName: uploadedFileName,
      sheetName: document.getElementById("sheetSelect").value,
      mode: currentMode,
    };

    if (currentMode === "create") {
      const tableName = document.getElementById("newTableName").value.trim();
      if (!tableName) throw new Error("Masukkan nama table Oracle.");

      const selectedColumns = [
        ...document.querySelectorAll(
          "#createColList input[type=checkbox]:checked",
        ),
      ].map((cb) => cb.value);

      if (selectedColumns.length === 0)
        throw new Error("Pilih minimal satu kolom.");

      payload.tableName = tableName;
      payload.selectedColumns = selectedColumns;
    } else {
      const tableName = document.getElementById("tableSelect").value;
      if (!tableName) throw new Error("Pilih table Oracle terlebih dahulu.");

      const columnMap = {};
      document.querySelectorAll(".column-map").forEach((sel) => {
        if (sel.value) {
          columnMap[sel.dataset.excel] = sel.value;
        }
      });

      if (Object.keys(columnMap).length === 0) {
        throw new Error("Pilih minimal satu mapping kolom.");
      }

      payload.tableName = tableName;
      payload.columnMap = columnMap;
    }

    const res = await fetch("/api/import", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });

    const result = await res.json();

    if (result.error) {
      resultBox.innerHTML = `
                <div class="result-error">
                    <strong>❌ Error:</strong> ${escHtml(result.error)}
                </div>
            `;
      return;
    }

    resultBox.innerHTML = `
            <div class="result-success">
                <div class="result-title">✅ Import Berhasil!</div>
                <div class="result-stats">
                    <span class="badge-success">✔ Inserted: ${result.inserted ?? 0}</span>
                    <span class="badge-warn">⏭ Skipped (duplikat): ${result.skipped ?? 0}</span>
                </div>
                <div class="result-msg">${escHtml(result.message ?? "")}</div>
            </div>
        `;
  } catch (err) {
    resultBox.innerHTML = `
            <div class="result-error">
                <strong>❌ Error:</strong> ${escHtml(err.message)}
            </div>
        `;
    console.error(err);
  }
}

// ============================================================
// Helpers
// ============================================================

// Use DOM to safely escape HTML (avoids formatter mangling entity strings)
function escHtml(str) {
  const el = document.createElement("span");
  el.appendChild(document.createTextNode(String(str)));
  return el.innerHTML;
}

// Escape double quotes for use inside HTML attribute values
function escAttr(str) {
  return escHtml(str).replace(/"/g, "&#34;");
}
