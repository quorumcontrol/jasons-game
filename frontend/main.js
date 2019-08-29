const { app, BrowserWindow, Menu } = require('electron');
const { spawn } = require('child_process');
const path = require('path');
const fs = require('fs');
const os = require('os');

if (require('electron-squirrel-startup')) {
    // Windows squirrel installer launches the app during install.
    // This cuts down on the weirdness from that.
    return app.quit();
}

let win;
let game;

function startGame() {
    runBackend();
    // TODO: Would be better if the backend signaled us when it was ready...
    setTimeout(createWindow, 3000);
}

function runBackendExecutable(path) {
    process.env.GOLOG_FILE = logPath(os.platform());

    game = spawn(path, [], {stdio: 'ignore'});

    game.on('error', (err) => {
        throw Error(`Error launching game backend: ${err}`);
    });

    game.on('exit', (code) => {
        if (code && code !== 0) {
            throw Error(`Game backend process exited with error code: ${code}`);
        }
    });
}

function runBackend() {
    let platform = process.platform;
    let exeSuffix = "";

    if (platform === "win32") {
        exeSuffix = ".exe";
    }

    let asarPath = path.resolve(__dirname, '..', '..', 'app.asar.unpacked', 'bin', `jasonsgame-${platform}-public${exeSuffix}`);
    let devPath = path.resolve(__dirname, '..', 'bin', `jasonsgame-${platform}-public${exeSuffix}`);

    fs.access(asarPath, fs.constants.F_OK, (err) => {
        if (err) {
            fs.access(devPath, fs.constants.F_OK, (devErr) => {
               if (devErr) {
                   throw(`Error finding game backend executable: ${err}`)
               }
               runBackendExecutable(devPath);
            });
        } else {
            runBackendExecutable(asarPath);
        }
    });
}

function createWindow () {
    win = new BrowserWindow({
        width: 1366,
        height: 768,
        backgroundColor: '#000000',
        webPreferences: {
            nodeIntegration: false
        }
    });

    win.loadURL('http://localhost:8080/');

    win.on('closed', () => {
        win = null;
    })
}

function logPath(platform) {
    let homeDir = os.homedir ? os.homedir() : process.env.HOME;

    let dir;
    switch (platform) {
        case 'darwin': {
            dir = path.join(homeDir, 'Library', 'Logs');
            break;
        }

        case 'win32': {
            dir = path.join(homeDir, 'AppData', 'Roaming');
            break;
        }

        default: {
            dir = path.join(homeDir, '.config', 'JasonsGame');
            break;
        }
    }

    return path.join(dir, 'jasonsgame.log');
}

function setupMenu() {
    if (process.platform === 'darwin') {
        let menuTemplate = [
            {
                label: app.getName(),
                submenu: [{
                    label: 'Quit',
                    accelerator: 'CmdOrCtrl+Q',
                    click: () => { app.quit(); }
                }]
            },
            {
                label: 'Edit',
                submenu: [
                    { role: 'cut' },
                    { role: 'copy' },
                    { role: 'paste' }
                ]
            },
            { role: 'viewMenu' },
            { role: 'windowMenu' },
            { role: 'helpMenu' }
        ];
        
        Menu.setApplicationMenu(Menu.buildFromTemplate(menuTemplate));
    }
}

app.on('ready', () => {
    console.log("Launching Jason's Game");

    setupMenu();

    startGame();
});

app.on('window-all-closed', () => {
    // On macOS it is common for applications and their menu bar
    // to stay active until the user quits explicitly with Cmd + Q
    if (process.platform !== 'darwin') {
        app.quit();
    }
});

app.on('before-quit', () => {
    if (win) {
        win.close();
    }

    if (game) {
        game.kill();
    }
});

app.on('activate', () => {
    // On macOS it's common to re-create a window in the app when the
    // dock icon is clicked and there are no other windows open.
    if (win === null) {
        createWindow();
    }
});
