const vscode = require('vscode');
const { LanguageClient } = require('vscode-languageclient/node');

let client;

function resolveServerOptions() {
  const cfg = vscode.workspace.getConfiguration('vox');
  const compilerPath = cfg.get('compilerPath', 'vox');
  const nodePath = cfg.get('nodePath', 'node');
  const env = { ...process.env, VOX_NODE: nodePath };
  return {
    run: { command: compilerPath, args: ['lsp'], options: { env } },
    debug: { command: compilerPath, args: ['lsp'], options: { env } },
  };
}

function resolveClientOptions() {
  return {
    documentSelector: [
      { scheme: 'file', language: 'vox' },
      { scheme: 'untitled', language: 'vox' },
    ],
    synchronize: {
      configurationSection: 'vox',
    },
  };
}

function activate(context) {
  const serverOptions = resolveServerOptions();
  const clientOptions = resolveClientOptions();
  client = new LanguageClient(
    'voxLsp',
    'Vox Language Server',
    serverOptions,
    clientOptions,
  );
  context.subscriptions.push(client.start());
}

function deactivate() {
  if (!client) return undefined;
  return client.stop();
}

module.exports = {
  activate,
  deactivate,
};
