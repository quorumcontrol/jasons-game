const { app, BrowserWindow, Menu } = require('electron');
const { spawn } = require('child_process');
const path = require('path');

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

function runBackend() {
    let platform = process.platform;
    let exeSuffix = "";

    if (platform === "win32") {
        exeSuffix = ".exe";
    }

    let executable = path.resolve(__dirname, '..', '..', 'app.asar.unpacked', 'bin', `jasonsgame-${platform}-public${exeSuffix}`);
    game = spawn(executable);

    game.on('error', (err) => {
        throw Error(`Error launching game backend: ${err}`);
    });

    game.on('exit', (code) => {
        if (code !== 0) {
            throw Error(`Game backend process exited with error code: ${code}`);
        }
    })
}

function createWindow () {
    win = new BrowserWindow({
        width: 1366,
        height: 768,
        backgroundColor: '#000000',
        webPreferences: {
            nodeIntegration: true
        }
    });

    win.loadURL('http://localhost:8080/');

    win.on('closed', () => {
        win = null;
    })
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
            }
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
