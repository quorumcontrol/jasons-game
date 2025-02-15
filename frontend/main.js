const { app, BrowserWindow, Menu, autoUpdater, ipcMain } = require('electron');
const { spawn } = require('child_process');
const path = require('path');
const fs = require('fs');
const os = require('os');
const log = require('electron-log');
const fetch = require('node-fetch');

if (require('electron-squirrel-startup')) {
    // Windows squirrel installer launches the app during install.
    // This cuts down on the weirdness from that.
    return
}

let win;

const gotTheLock = app.requestSingleInstanceLock();

if (!gotTheLock) {
    log.info('Quitting because an instance is already running');
    app.quit();
} else {
    app.on('second-instance', (event, commandLine, workingDirectory) => {
        // someone tried to run a second instance, we should focus our window
        if (win) {
            if (win.isMinimized()) {
                win.restore();
            }
            win.focus();
        }
    });
}

const backendURL = 'http://localhost:8080/';
const updateFile = path.resolve(__dirname, 'update.html');
const updateFileLinux = path.resolve(__dirname, 'update_linux.html');
const restartFile = path.resolve(__dirname, 'restart.html');

if (process.env.JGDEBUG) {
    log.transports.file.level = 'debug';
} else {
    log.transports.file.level = 'info';
}

let game;
let gameKilled = false;
let updateAvailable = false;
let quitting = false;

const autoUpdateFeedURL = `https://hazel.quorumcontrol.now.sh/update/${process.platform}/${app.getVersion()}`;

autoUpdater.setFeedURL({'url': autoUpdateFeedURL });

autoUpdater.on('error', (message) => {
    log.error('There was a problem updating the application');
    log.error(message);
});

function killGame() {
    log.info('Killing game backend process');
    gameKilled = true;
    if (game) {
        game.kill();
    }
}

function updateAvailableHandler() {
    updateAvailable = true;

    if (game) {
        killGame();
    }

    log.info('Update is available');

    if (win) {
        log.info('Loading update into existing window');
        if (process.platform === 'linux') {
            win.loadFile(updateFileLinux);
        } else {
            win.loadFile(updateFile);
        }
    } else {
        // Presumably we haven't created it yet and will notice the
        // updateAvailable flag when we do. Trying to create one here
        // got pretty race-y.
        log.info('No existing window found for update');
    }
}

autoUpdater.on('update-available', updateAvailableHandler);

autoUpdater.on('checking-for-update', () => {
    log.info(`Checking for update (current version is ${app.getVersion()})`);
});

autoUpdater.on('update-not-available', () => {
    log.info('No update available');
});

autoUpdater.on('update-downloaded', () => {
    log.info('Update has been downloaded');

    win.loadFile(restartFile);
});

// Linux isn't supported by the built-in autoUpdater 🙄
function checkForUpdatesLinux() {
    log.info(`Checking for update (current version is ${app.getVersion()})`);

    fetch(autoUpdateFeedURL).then(response => {
        if (response.status === 200) {
            updateAvailableHandler();
        }
    });
}

function checkForUpdates() {
    if (process.platform === 'linux') {
        checkForUpdatesLinux();
    } else {
        autoUpdater.checkForUpdates();
    }
}

function startUpdater() {
    checkForUpdates();

    setInterval(() => {
        checkForUpdates();
    }, 15 * 60 * 1000);
}

function startGame() {
    runBackend();
    // TODO: Would be better if the backend signaled us when it was ready...
    setTimeout(() => { if (!win) { createMainWindow() } }, 1000);
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

        if (process.env.JGDEBUG) {
            win.webContents.openDevTools();
        }

    });

    win.on('close', () => {
        log.debug('window event: close');

        if (quitting) {
            win = null;
        }
    });

    win.on('closed', () => {
        log.debug('window event: closed');

        win = null;
    })
}

function createMainWindow() {
    log.info(`Creating main window (updateAvailable: ${updateAvailable})`);

    if (updateAvailable) {
        if (process.platform === 'linux') {
            createWindow(null, updateFileLinux);
        } else {
            createWindow(null, updateFile);
        }
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

app.on('ready', () => {
    log.debug('event: ready');

    setupMenu();

    startUpdater();

    startGame();
});

app.on('window-all-closed', () => {
    log.debug('event: window-all-closed');

    // On macOS it is common for applications and their menu bar
    // to stay active until the user quits explicitly with Cmd + Q
    if (quitting || process.platform !== 'darwin') {
        app.quit();
    }
});

function shutdown() {
    quitting = true;

    if (win) {
        win.close();
    }

    if (game) {
        killGame();
    }
}

app.on('before-quit', () => {
    log.debug('event: before-quit');

    shutdown();
});

autoUpdater.on('before-quit-for-update', () => {
    log.debug('event: before-quit-for-update');

    shutdown();
});

app.on('activate', () => {
    log.debug('event: activate');

    // On macOS it's common to re-create a window in the app when the
    // dock icon is clicked and there are no other windows open.
    if (win === null) {
        createMainWindow();
    }
});

ipcMain.on('auto-updater', (event, message) => {
    if (message === 'quitAndInstall') {
        autoUpdater.quitAndInstall();
    }
});
