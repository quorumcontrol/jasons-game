const { app, BrowserWindow, Menu, autoUpdater } = require('electron');
const { spawn } = require('child_process');
const path = require('path');
const fs = require('fs');
const os = require('os');
const log = require('electron-log');

const backendURL = 'http://localhost:8080/';
const updateFile = path.resolve(__dirname, 'update.html');

if (require('electron-squirrel-startup')) {
    // Windows squirrel installer launches the app during install.
    // This cuts down on the weirdness from that.
    return app.quit();
}

let win;
let game;
let gameKilled = false;
let updating = false;

autoUpdater.setFeedURL({'url': `https://hazel.quorumcontrol.now.sh/update/${process.platform}/${app.getVersion()}`});

autoUpdater.on('error', (message) => {
    log.error('There was a problem updating the application');
    log.error(message);
});

function killGame() {
    log.info('Killing game backend process');
    gameKilled = true;
    game.kill();
}

autoUpdater.on('update-available', () => {
    updating = true;

    if (game) {
        killGame();
    }

    log.info('Update is available');

    if (win) {
        log.info('Loading update into existing window');
        win.loadFile(updateFile);
    } else {
        createMainWindow();
    }
});

autoUpdater.on('checking-for-update', () => {
    log.info('Checking for update');
});

autoUpdater.on('update-not-available', () => {
    log.info('No update available');
});

function startUpdater() {
    autoUpdater.checkForUpdates();

    setInterval(() => {
        autoUpdater.checkForUpdates();
    }, 15 * 60 * 1000);
}

function startGame() {
    runBackend();
    // TODO: Would be better if the backend signaled us when it was ready...
    setTimeout(() => { createMainWindow() }, 1000);
}

function runBackendExecutable(exePath) {
    process.env.GOLOG_FILE = path.join(logPath(os.platform()), 'jasonsgame.log');

    game = spawn(exePath, [], {stdio: 'ignore'});

    game.on('error', (err) => {
        throw Error(`Error launching game backend: ${err}`);
    });

    game.on('exit', (code) => {
        if (!gameKilled && code && code !== 0) {
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

function createWindow (url, file) {
    win = new BrowserWindow({
        width: 1366,
        height: 768,
        backgroundColor: '#000000',
        show: false,
        webPreferences: {
            nodeIntegration: false,
            preload: path.resolve(__dirname, 'preload.js')
        },
    });

    if (url) {
        log.info(`Loading URL ${url}`);
        win.loadURL(url);
    } else {
        log.info(`Loading file ${file}`);
        win.loadFile(file);
    }

    win.once('ready-to-show', () => {
        win.show();

        if (process.env.DEBUG) {
            win.webContents.openDevTools();
        }

    });

    win.on('closed', () => {
        win = null;
    })
}

function createMainWindow() {
    if (updating) {
        createWindow(null, updateFile);
    } else {
        createWindow(backendURL);
    }
}

function logPath(platform) {
    let homeDir = os.homedir ? os.homedir() : process.env.HOME;

    switch (platform) {
        case 'darwin': {
            return path.join(homeDir, 'Library', 'Logs');
        }

        case 'win32': {
            return path.join(homeDir, 'AppData', 'Roaming');
        }

        default: {
            return path.join(homeDir, '.config', 'JasonsGame');
        }
    }
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

app.on('error', (err) => {
    log.error(`application error: ${err}`);
});

app.on('ready', () => {
    setupMenu();

    startUpdater();

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
        killGame();
    }
});

app.on('activate', () => {
    // On macOS it's common to re-create a window in the app when the
    // dock icon is clicked and there are no other windows open.
    if (win === null) {
        createMainWindow();
    }
});
