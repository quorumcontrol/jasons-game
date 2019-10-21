const { shell } = require('electron');

window.openExternal = function(url) {
    shell.openExternal(url);
};
