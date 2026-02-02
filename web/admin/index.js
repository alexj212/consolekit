const loginForm = document.getElementById('login-form');
const terminalWrapper = document.getElementById('terminal-wrapper');
const terminalDiv = document.getElementById('terminal');
const errorDiv = document.getElementById('error');

// Update connection status indicator
function updateStatus(connected) {
    const statusDot = document.querySelector('.status-dot');
    const statusText = document.getElementById('status-text');

    if (!statusDot || !statusText) return;

    if (connected) {
        statusDot.classList.remove('disconnected');
        statusDot.classList.add('connected');
        statusText.textContent = 'Connected';
    } else {
        statusDot.classList.remove('connected');
        statusDot.classList.add('disconnected');
        statusText.textContent = 'Disconnected';
    }
}

let input = "";
let cursorPos = 0;  // Current cursor position within input
let history = [];
let historyIndex = -1;
let term = null;
let socket = null;

// Application configuration
let appConfig = {
    appName: "ConsoleKit",
    pageTitle: "ConsoleKit Web Service",
    welcome: "Welcome to ConsoleKit Web Terminal",
    motd: ""
};

// Load configuration from server
async function loadConfig() {
    try {
        const response = await fetch("/config");
        if (response.ok) {
            appConfig = await response.json();
            // Update page title
            document.title = appConfig.pageTitle;
        }
    } catch (e) {
        console.error("Failed to load config:", e);
    }
}

window.login = function login() {
    const username = document.getElementById('username').value;
    const password = document.getElementById('password').value;

    errorDiv.textContent = "";

    fetch("/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username, password }),
    })
        .then((res) => {
            if (res.ok) {
                loginForm.style.display = "none";
                terminalWrapper.style.display = "flex";
                loadConfig().then(() => startTerminal());
            } else {
                errorDiv.textContent = "Invalid credentials.";
            }
        })
        .catch((err) => {
            console.error("Login request failed:", err);
            errorDiv.textContent = "Login request failed. Please try again.";
        });
};

window.logout = function logout() {
    fetch("/logout", {
        method: "POST",
        credentials: "include",
    }).finally(() => {
        cleanupAndShowLogin();
    });
};

function clearLine(term) {
    term.write("\x1b[2K\r$ ");
}

// Redraw the current input line with cursor at correct position
function redrawLine(term) {
    term.write("\x1b[2K\r$ " + input);
    // Move cursor to correct position
    const moveBack = input.length - cursorPos;
    if (moveBack > 0) {
        term.write("\x1b[" + moveBack + "D");
    }
}

// Insert character at cursor position
function insertChar(char) {
    input = input.slice(0, cursorPos) + char + input.slice(cursorPos);
    cursorPos++;
}

// Delete character at cursor position (backspace)
function deleteCharBefore() {
    if (cursorPos > 0) {
        input = input.slice(0, cursorPos - 1) + input.slice(cursorPos);
        cursorPos--;
        return true;
    }
    return false;
}

// Delete character ahead of cursor (delete key)
function deleteCharAhead() {
    if (cursorPos < input.length) {
        input = input.slice(0, cursorPos) + input.slice(cursorPos + 1);
        return true;
    }
    return false;
}

// Delete from cursor to end of line
function deleteToEnd() {
    if (cursorPos < input.length) {
        input = input.slice(0, cursorPos);
        return true;
    }
    return false;
}

// Delete from start to cursor
function deleteToStart() {
    if (cursorPos > 0) {
        input = input.slice(cursorPos);
        cursorPos = 0;
        return true;
    }
    return false;
}

// Delete word before cursor
function deleteWordBefore() {
    if (cursorPos === 0) return false;

    let pos = cursorPos - 1;
    // Skip trailing spaces
    while (pos >= 0 && input[pos] === ' ') pos--;
    // Delete word
    while (pos >= 0 && input[pos] !== ' ') pos--;

    const newPos = pos + 1;
    input = input.slice(0, newPos) + input.slice(cursorPos);
    cursorPos = newPos;
    return true;
}

// Set input and reset cursor
function setInput(newInput) {
    input = newInput;
    cursorPos = input.length;
}

// Load history from localStorage
function loadHistory() {
    try {
        const saved = localStorage.getItem('consolekit_history');
        if (saved) {
            return JSON.parse(saved);
        }
    } catch (e) {
        console.error("Failed to load history:", e);
    }
    return [];
}

// Save history to localStorage
function saveHistory() {
    try {
        // Keep only last 1000 commands
        const toSave = history.slice(-1000);
        localStorage.setItem('consolekit_history', JSON.stringify(toSave));
    } catch (e) {
        console.error("Failed to save history:", e);
    }
}

function cleanupAndShowLogin(message = "Session disconnected.") {
    if (socket) {
        socket.close();
        socket = null;
    }
    if (term) {
        term.dispose();
        term = null;
    }
    terminalDiv.innerHTML = '';
    terminalWrapper.style.display = "none";
    loginForm.style.display = "flex";
    errorDiv.textContent = message;
    input = "";
    cursorPos = 0;
    history = [];
    historyIndex = -1;
}

function startTerminal() {
    term = new Terminal({
        theme: {
            background: '#000000',
            foreground: '#00ff00',
            cursor: '#00ff00',
            selection: '#336633',
        },
        cursorBlink: true,
        cursorStyle: 'block',
        fontFamily: '"Cascadia Code", "Fira Code", "Consolas", "Monaco", monospace',
        fontSize: 14,
        lineHeight: 1.2,
        letterSpacing: 0,
        scrollback: 10000,
        tabStopWidth: 4,
        allowProposedApi: true,
    });

    const fitAddon = new FitAddon.FitAddon();
    term.loadAddon(fitAddon);

    const webLinksAddon = new WebLinksAddon.WebLinksAddon();
    term.loadAddon(webLinksAddon);

    term.open(terminalDiv);
    fitAddon.fit();
    term.focus();

    input = "";
    cursorPos = 0;
    history = loadHistory();  // Load persistent history
    historyIndex = history.length;

    try {
        socket = new WebSocket("ws://" + location.host + "/repl");
    } catch (err) {
        cleanupAndShowLogin("Failed to connect to server.");
        return;
    }

    socket.onopen = function() {
        updateStatus(true);
    };

    term.write(appConfig.welcome + "\r\n");
    term.write("=".repeat(appConfig.welcome.length) + "\r\n\r\n");

    if (appConfig.motd) {
        // Convert \n to \r\n for proper terminal display
        const motd = appConfig.motd.replace(/\n/g, "\r\n");
        term.write(motd + "\r\n\r\n");
    }

    term.write("Keyboard Shortcuts:\r\n");
    term.write("  Ctrl+A/Home    - Beginning of line\r\n");
    term.write("  Ctrl+E/End     - End of line\r\n");
    term.write("  Ctrl+U         - Clear line before cursor\r\n");
    term.write("  Ctrl+K         - Clear line after cursor\r\n");
    term.write("  Ctrl+W         - Delete word before cursor\r\n");
    term.write("  Ctrl+L         - Clear screen\r\n");
    term.write("  Ctrl+C         - Cancel current input\r\n");
    term.write("  Ctrl+D         - Logout (on empty line)\r\n");
    term.write("  Up/Down        - Command history\r\n");
    term.write("  Left/Right     - Move cursor\r\n\r\n");
    term.write("Type 'help' for command help, 'exit' or 'quit' to logout\r\n\r\n$ ");

    socket.onmessage = function (event) {
        try {
            const msg = JSON.parse(event.data);
            if (msg.type === "output") {
                const output = msg.message.replace(/\n/g, "\r\n");
                term.write(output + "\r\n$ ");
            } else if (msg.type === "error") {
                const errorMsg = msg.message.replace(/\n/g, "\r\n");
                term.write("\r\n[Error] " + errorMsg + "\r\n$ ");
            }
        } catch (e) {
            term.write("\r\n[Invalid JSON received]\r\n$ ");
        }
    };

    socket.onclose = function () {
        updateStatus(false);
        term.write("\r\n[Session closed]\r\n");
        setTimeout(() => cleanupAndShowLogin("Connection lost."), 1000);
    };

    socket.onerror = function () {
        updateStatus(false);
        term.write("\r\n[WebSocket error]\r\n");
        setTimeout(() => cleanupAndShowLogin("WebSocket connection error."), 1000);
    };

    window.addEventListener("resize", () => {
        fitAddon.fit();
    });

    // Handle paste events
    term.onData((data) => {
        // If data contains multiple characters, it's likely a paste
        if (data.length > 1) {
            // Insert pasted text at cursor position
            for (let i = 0; i < data.length; i++) {
                const char = data[i];
                // Only insert printable characters (ignore control characters)
                if (char >= " " && char <= "~") {
                    insertChar(char);
                }
            }
            redrawLine(term);
        }
    });

    term.onKey(({ key, domEvent }) => {
        const code = domEvent.code;

        // Handle Ctrl key combinations
        if (domEvent.ctrlKey) {
            switch (key) {
                case "a":  // Ctrl+A: Move to beginning of line
                case "A":
                    domEvent.preventDefault();
                    cursorPos = 0;
                    redrawLine(term);
                    return;

                case "e":  // Ctrl+E: Move to end of line
                case "E":
                    domEvent.preventDefault();
                    cursorPos = input.length;
                    redrawLine(term);
                    return;

                case "u":  // Ctrl+U: Clear from start to cursor
                case "U":
                    domEvent.preventDefault();
                    if (deleteToStart()) {
                        redrawLine(term);
                    }
                    return;

                case "k":  // Ctrl+K: Clear from cursor to end
                case "K":
                    domEvent.preventDefault();
                    if (deleteToEnd()) {
                        redrawLine(term);
                    }
                    return;

                case "w":  // Ctrl+W: Delete word before cursor
                case "W":
                    domEvent.preventDefault();
                    if (deleteWordBefore()) {
                        redrawLine(term);
                    }
                    return;

                case "l":  // Ctrl+L: Clear screen
                case "L":
                    domEvent.preventDefault();
                    term.clear();
                    term.write("$ " + input);
                    const moveBack = input.length - cursorPos;
                    if (moveBack > 0) {
                        term.write("\x1b[" + moveBack + "D");
                    }
                    return;

                case "c":  // Ctrl+C: Cancel current input
                case "C":
                    domEvent.preventDefault();
                    term.write("^C\r\n$ ");
                    input = "";
                    cursorPos = 0;
                    historyIndex = history.length;
                    return;

                case "d":  // Ctrl+D: Logout if line is empty
                case "D":
                    domEvent.preventDefault();
                    if (input.length === 0) {
                        term.write("logout\r\n");
                        socket.close();
                        logout();
                    } else {
                        // Delete character ahead if line has content
                        if (deleteCharAhead()) {
                            redrawLine(term);
                        }
                    }
                    return;

                default:
                    return;
            }
        }

        switch (code) {
            case "Enter":
                term.write("\r\n");
                const trimmed = input.trim();
                if (trimmed !== "") {
                    history.push(trimmed);
                    saveHistory();  // Persist history to localStorage
                    historyIndex = history.length;

                    if (trimmed === "exit" || trimmed === "quit") {
                        term.write("Logging out...\r\n");
                        socket.close();
                        logout();
                        return;
                    }

                    const payload = {
                        type: "input",
                        message: trimmed,
                    };

                    try {
                        socket.send(JSON.stringify(payload));
                    } catch (e) {
                        term.write("[Send error: disconnected]\r\n");
                        cleanupAndShowLogin("Lost connection during command.");
                        return;
                    }
                } else {
                    // Empty input - just show prompt again
                    term.write("$ ");
                }
                input = "";
                cursorPos = 0;
                break;

            case "Backspace":
                if (deleteCharBefore()) {
                    redrawLine(term);
                }
                break;

            case "Delete":
                if (deleteCharAhead()) {
                    redrawLine(term);
                }
                break;

            case "ArrowLeft":
                if (cursorPos > 0) {
                    cursorPos--;
                    term.write("\x1b[D");  // Move cursor left
                }
                break;

            case "ArrowRight":
                if (cursorPos < input.length) {
                    cursorPos++;
                    term.write("\x1b[C");  // Move cursor right
                }
                break;

            case "ArrowUp":
                if (historyIndex > 0) {
                    historyIndex--;
                    setInput(history[historyIndex]);
                    redrawLine(term);
                }
                break;

            case "ArrowDown":
                if (historyIndex < history.length - 1) {
                    historyIndex++;
                    setInput(history[historyIndex]);
                    redrawLine(term);
                } else {
                    historyIndex = history.length;
                    setInput("");
                    redrawLine(term);
                }
                break;

            case "Home":
                cursorPos = 0;
                redrawLine(term);
                break;

            case "End":
                cursorPos = input.length;
                redrawLine(term);
                break;

            case "Tab":
                domEvent.preventDefault();
                // Simple command completion (could be enhanced with server-side completion)
                if (input.trim().length > 0) {
                    // Get list of common commands
                    const commands = [
                        "help", "exit", "quit", "cls", "clear", "print", "date",
                        "history", "alias", "jobs", "vars", "config", "log",
                        "cat", "grep", "env", "http", "sleep", "template", "schedule"
                    ];

                    const words = input.trim().split(/\s+/);
                    const currentWord = words[words.length - 1];

                    // Find matching commands
                    const matches = commands.filter(cmd => cmd.startsWith(currentWord));

                    if (matches.length === 1) {
                        // Complete the command
                        const completion = matches[0].substring(currentWord.length);
                        for (let i = 0; i < completion.length; i++) {
                            insertChar(completion[i]);
                        }
                        redrawLine(term);
                    } else if (matches.length > 1) {
                        // Show all matches
                        term.write("\r\n");
                        term.write(matches.join("  ") + "\r\n");
                        term.write("$ " + input);
                        const moveBack = input.length - cursorPos;
                        if (moveBack > 0) {
                            term.write("\x1b[" + moveBack + "D");
                        }
                    } else {
                        // No matches - insert tab as spaces
                        for (let i = 0; i < 4; i++) {
                            insertChar(" ");
                        }
                        redrawLine(term);
                    }
                } else {
                    // Empty line - insert tab as spaces
                    for (let i = 0; i < 4; i++) {
                        insertChar(" ");
                    }
                    redrawLine(term);
                }
                break;

            default:
                // Insert printable characters at cursor position
                if (key.length === 1 && key >= " " && !domEvent.altKey && !domEvent.metaKey) {
                    insertChar(key);
                    redrawLine(term);
                }
                break;
        }
    });
}

loginForm.style.display = "flex";
