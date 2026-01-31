const loginForm = document.getElementById('login-form');
const terminalWrapper = document.getElementById('terminal-wrapper');
const terminalDiv = document.getElementById('terminal');
const errorDiv = document.getElementById('error');

let input = "";
let history = [];
let historyIndex = -1;
let term = null;
let socket = null;

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
                startTerminal();
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
    history = [];
    historyIndex = -1;
}

function startTerminal() {
    term = new Terminal({
        theme: {
            background: '#000000',
            foreground: '#00ff00',
        },
        cursorBlink: true,
        fontFamily: 'monospace',
        fontSize: 14,
    });

    const fitAddon = new FitAddon.FitAddon();
    term.loadAddon(fitAddon);

    const webLinksAddon = new WebLinksAddon.WebLinksAddon();
    term.loadAddon(webLinksAddon);

    term.open(terminalDiv);
    fitAddon.fit();
    term.focus();

    input = "";
    history = [];
    historyIndex = -1;

    try {
        socket = new WebSocket("ws://" + location.host + "/repl");
    } catch (err) {
        cleanupAndShowLogin("Failed to connect to server.");
        return;
    }

    term.write("Welcome to ConsoleKit Web Terminal\r\n\r\n$ ");

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
        term.write("\r\n[Session closed]\r\n");
        cleanupAndShowLogin("Connection lost.");
    };

    socket.onerror = function () {
        term.write("\r\n[WebSocket error]\r\n");
        cleanupAndShowLogin("WebSocket connection error.");
    };

    window.addEventListener("resize", () => {
        fitAddon.fit();
    });

    term.onKey(({ key, domEvent }) => {
        const code = domEvent.code;

        switch (code) {
            case "Enter":
                term.write("\r\n");
                const trimmed = input.trim();
                if (trimmed !== "") {
                    history.push(trimmed);
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
                }
                input = "";
                term.write("$ ");
                break;

            case "Backspace":
                if (input.length > 0) {
                    term.write("\b \b");
                    input = input.slice(0, -1);
                }
                break;

            case "ArrowUp":
                if (historyIndex > 0) {
                    historyIndex--;
                    clearLine(term);
                    input = history[historyIndex];
                    term.write(input);
                }
                break;

            case "ArrowDown":
                if (historyIndex < history.length - 1) {
                    historyIndex++;
                    clearLine(term);
                    input = history[historyIndex];
                    term.write(input);
                } else {
                    historyIndex = history.length;
                    clearLine(term);
                    input = "";
                }
                break;

            default:
                if (key >= " " && !domEvent.ctrlKey && !domEvent.metaKey) {
                    term.write(key);
                    input += key;
                }
                break;
        }
    });
}

loginForm.style.display = "flex";
