document.addEventListener('DOMContentLoaded', () => {
    // --- State ---
    const state = {
        width: 10,
        height: 10,
        maxSteps: 1000,
        grid: [], // 2D array: true = alive, false = dead
        isDrawing: false,
        drawMode: true // true = paint alive, false = paint dead
    };

    // --- DOM Elements ---
    const els = {
        width: document.getElementById('width'),
        height: document.getElementById('height'),
        maxSteps: document.getElementById('max-steps'),
        applyDims: document.getElementById('apply-dims'),
        grid: document.getElementById('grid'),
        gridOverlay: document.getElementById('grid-overlay'),
        confirmResize: document.getElementById('confirm-resize'),
        cancelResize: document.getElementById('cancel-resize'),
        clearBtn: document.getElementById('clear-btn'),
        fillBtn: document.getElementById('fill-btn'),
        invertBtn: document.getElementById('invert-btn'),
        randomBtn: document.getElementById('random-btn'),
        density: document.getElementById('density'),
        liveCount: document.getElementById('live-count'),
        totalCount: document.getElementById('total-count'),
        hexInput: document.getElementById('hex-input'),
        loadHexBtn: document.getElementById('load-hex-btn'),
        hexError: document.getElementById('hex-error'),
        cmdOutput: document.getElementById('command-output'),
        copyCmdBtn: document.getElementById('copy-cmd-btn'),
        maskOutput: document.getElementById('bitmask-output'),
        copyMaskBtn: document.getElementById('copy-mask-btn'),
        toast: document.getElementById('toast')
    };

    // --- Initialization ---
    function init() {
        loadState();
        renderGrid();
        updateStats();
        updateOutput();
        setupEventListeners();
    }

    // --- Grid Logic ---
    function createGrid(w, h) {
        const newGrid = [];
        for (let y = 0; y < h; y++) {
            newGrid.push(new Array(w).fill(false));
        }
        return newGrid;
    }

    function resizeGrid(w, h) {
        state.width = w;
        state.height = h;
        state.grid = createGrid(w, h);
        renderGrid();
        updateStats();
        updateOutput();
        saveState();
    }

    function renderGrid() {
        els.grid.innerHTML = '';
        els.grid.style.gridTemplateColumns = `repeat(${state.width}, 20px)`;
        
        const fragment = document.createDocumentFragment();
        
        for (let y = 0; y < state.height; y++) {
            for (let x = 0; x < state.width; x++) {
                const cell = document.createElement('div');
                cell.className = 'cell';
                if (state.grid[y][x]) cell.classList.add('alive');
                cell.dataset.x = x;
                cell.dataset.y = y;
                fragment.appendChild(cell);
            }
        }
        
        els.grid.appendChild(fragment);
    }

    function updateCellVisual(x, y) {
        const index = y * state.width + x;
        const cell = els.grid.children[index];
        if (state.grid[y][x]) {
            cell.classList.add('alive');
        } else {
            cell.classList.remove('alive');
        }
    }

    // --- Bitmask Encoding ---
    // Convention: Row-major, 4-bit nibbles, first bit is MSB.
    function generateBitmask() {
        let bits = '';
        for (let y = 0; y < state.height; y++) {
            for (let x = 0; x < state.width; x++) {
                bits += state.grid[y][x] ? '1' : '0';
            }
        }

        // Pad to multiple of 4
        const remainder = bits.length % 4;
        if (remainder !== 0) {
            bits += '0'.repeat(4 - remainder);
        }

        let hex = '';
        for (let i = 0; i < bits.length; i += 4) {
            const nibble = bits.substr(i, 4);
            // parseInt(nibble, 2) treats string as binary.
            // '1000' -> 8. This matches "first bit is MSB".
            hex += parseInt(nibble, 2).toString(16).toUpperCase();
        }

        return hex || '0'; // Default to 0 if empty (though grid always has size)
    }

    function loadFromBitmask(hex) {
        // Validate hex
        if (!/^[0-9A-Fa-f]+$/.test(hex)) {
            throw new Error("Invalid HEX characters");
        }

        // Convert hex to binary string
        let bits = '';
        for (let char of hex) {
            const val = parseInt(char, 16);
            // Convert to 4-bit binary, MSB first
            bits += val.toString(2).padStart(4, '0');
        }

        const totalCells = state.width * state.height;
        // We allow bits string to be longer due to padding, but not shorter
        // Actually, user might paste a partial mask, but let's enforce strictness or just fill what we can.
        // Requirement says "ignore extra padded bits".
        
        // Reset grid
        const newGrid = createGrid(state.width, state.height);
        
        let bitIndex = 0;
        for (let y = 0; y < state.height; y++) {
            for (let x = 0; x < state.width; x++) {
                if (bitIndex < bits.length) {
                    newGrid[y][x] = bits[bitIndex] === '1';
                    bitIndex++;
                }
            }
        }

        state.grid = newGrid;
        renderGrid(); // Full re-render needed
        updateStats();
        updateOutput();
        saveState();
    }

    // --- UI Updates ---
    function updateStats() {
        let live = 0;
        for (let row of state.grid) {
            for (let cell of row) {
                if (cell) live++;
            }
        }
        els.liveCount.textContent = live;
        els.totalCount.textContent = state.width * state.height;
    }

    function updateOutput() {
        const mask = generateBitmask();
        const cmd = `./cgol_runner --box ${state.width} ${state.height} --bitmask ${mask} --max-steps ${state.maxSteps}`;
        
        els.cmdOutput.textContent = cmd;
        els.maskOutput.textContent = mask;
    }

    function showToast() {
        els.toast.classList.remove('hidden');
        setTimeout(() => {
            els.toast.classList.add('hidden');
        }, 2000);
    }

    // --- Persistence ---
    function saveState() {
        const data = {
            width: state.width,
            height: state.height,
            maxSteps: state.maxSteps,
            grid: state.grid
        };
        localStorage.setItem('cgol_designer_state', JSON.stringify(data));
    }

    function loadState() {
        const saved = localStorage.getItem('cgol_designer_state');
        if (saved) {
            try {
                const data = JSON.parse(saved);
                state.width = data.width || 10;
                state.height = data.height || 10;
                state.maxSteps = data.maxSteps || 1000;
                state.grid = data.grid || createGrid(state.width, state.height);
                
                // Validate grid dimensions match
                if (state.grid.length !== state.height || state.grid[0].length !== state.width) {
                    state.grid = createGrid(state.width, state.height);
                }
            } catch (e) {
                console.error("Failed to load state", e);
                state.grid = createGrid(10, 10);
            }
        } else {
            state.grid = createGrid(state.width, state.height);
        }

        // Sync inputs
        els.width.value = state.width;
        els.height.value = state.height;
        els.maxSteps.value = state.maxSteps;
    }

    // --- Event Listeners ---
    function setupEventListeners() {
        // Resize
        els.applyDims.addEventListener('click', () => {
            const w = parseInt(els.width.value);
            const h = parseInt(els.height.value);
            
            if (w === state.width && h === state.height) return;
            
            // Check if grid has live cells
            const hasLive = state.grid.some(row => row.some(c => c));
            
            if (hasLive) {
                els.gridOverlay.classList.remove('hidden');
            } else {
                resizeGrid(w, h);
            }
        });

        els.confirmResize.addEventListener('click', () => {
            const w = parseInt(els.width.value);
            const h = parseInt(els.height.value);
            resizeGrid(w, h);
            els.gridOverlay.classList.add('hidden');
        });

        els.cancelResize.addEventListener('click', () => {
            els.gridOverlay.classList.add('hidden');
            els.width.value = state.width;
            els.height.value = state.height;
        });

        // Max Steps
        els.maxSteps.addEventListener('change', () => {
            state.maxSteps = parseInt(els.maxSteps.value);
            updateOutput();
            saveState();
        });

        // Grid Interaction (Delegation)
        els.grid.addEventListener('mousedown', (e) => {
            if (e.target.classList.contains('cell')) {
                e.preventDefault(); // Prevent text selection
                state.isDrawing = true;
                
                const x = parseInt(e.target.dataset.x);
                const y = parseInt(e.target.dataset.y);
                
                // Determine mode: Left click = toggle/paint alive, Right/Shift = paint dead
                if (e.button === 2 || e.shiftKey) {
                    state.drawMode = false;
                } else {
                    // If clicking a single cell, toggle it. 
                    // For dragging, we'll assume painting alive unless it was already alive?
                    // Standard paint behavior: click sets to opposite, drag continues that state.
                    // Let's simplify: Click toggles. Drag paints the target state of the first click.
                    state.drawMode = !state.grid[y][x];
                }
                
                applyPaint(x, y);
            }
        });

        els.grid.addEventListener('mouseover', (e) => {
            if (state.isDrawing && e.target.classList.contains('cell')) {
                const x = parseInt(e.target.dataset.x);
                const y = parseInt(e.target.dataset.y);
                applyPaint(x, y);
            }
        });

        document.addEventListener('mouseup', () => {
            state.isDrawing = false;
        });

        // Prevent context menu on grid
        els.grid.addEventListener('contextmenu', (e) => e.preventDefault());

        function applyPaint(x, y) {
            if (state.grid[y][x] !== state.drawMode) {
                state.grid[y][x] = state.drawMode;
                updateCellVisual(x, y);
                updateStats();
                updateOutput();
                saveState();
            }
        }

        // Tools
        els.clearBtn.addEventListener('click', () => {
            state.grid = createGrid(state.width, state.height);
            renderGrid();
            updateStats();
            updateOutput();
            saveState();
        });

        els.fillBtn.addEventListener('click', () => {
            for (let y = 0; y < state.height; y++) {
                state.grid[y].fill(true);
            }
            renderGrid();
            updateStats();
            updateOutput();
            saveState();
        });

        els.invertBtn.addEventListener('click', () => {
            for (let y = 0; y < state.height; y++) {
                for (let x = 0; x < state.width; x++) {
                    state.grid[y][x] = !state.grid[y][x];
                }
            }
            renderGrid();
            updateStats();
            updateOutput();
            saveState();
        });

        els.randomBtn.addEventListener('click', () => {
            const density = parseInt(els.density.value) / 100;
            for (let y = 0; y < state.height; y++) {
                for (let x = 0; x < state.width; x++) {
                    state.grid[y][x] = Math.random() < density;
                }
            }
            renderGrid();
            updateStats();
            updateOutput();
            saveState();
        });

        // Hex Import
        els.loadHexBtn.addEventListener('click', () => {
            const hex = els.hexInput.value.trim();
            els.hexError.textContent = '';
            if (!hex) return;
            
            try {
                loadFromBitmask(hex);
                els.hexInput.value = '';
            } catch (e) {
                els.hexError.textContent = e.message;
            }
        });

        // Copy Buttons
        els.copyCmdBtn.addEventListener('click', () => {
            navigator.clipboard.writeText(els.cmdOutput.textContent).then(showToast);
        });

        els.copyMaskBtn.addEventListener('click', () => {
            navigator.clipboard.writeText(els.maskOutput.textContent).then(showToast);
        });
    }

    // Start
    init();
});
