const { ipcRenderer, remote, shell } = require('electron');

const _ipcRenderer = ipcRenderer;
const _appVersion = remote.app.getVersion();
const _openExternal = shell.openExternal;

process.once('loaded', () => {
    global.autoUpdaterQuitAndInstall = () => {
        _ipcRenderer.send('auto-updater', 'quitAndInstall');
    };

    global.appVersion = _appVersion;

    global.openExternal = _openExternal;
});
