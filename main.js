const { app, BrowserWindow } = require('electron');
const { spawn } = require('child_process');
const path = require('path');

let win;
let game;

function startGame() {
    runBackend();
    setTimeout(createWindow, 3000);
}

function runBackend() {
    let platform = process.platform;
    let executable = path.resolve(__dirname, 'bin', 'jasonsgame-' + platform + '-public');
    game = spawn(executable, [], {stdio: 'inherit'});
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

app.on('ready', () => {
    console.log("Launching Jason's Game");
    startGame();
});

app.on('window-all-closed', () => {
    // On macOS it is common for applications and their menu bar
    // to stay active until the user quits explicitly with Cmd + Q
    if (process.platform !== 'darwin') {
        app.quit();
    }
});

app.on('quit', () => {
    if (game !== null) {
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
