"use strict";

const fs = require("node:fs");
const path = require("node:path");
const vscode = require("vscode");

const COMMAND_ID = "ahdcode.runFile";
const MISSING_EXECUTABLE_MESSAGE =
  "AhdCode executable was not found on PATH.\n" +
  "Install AhdCode or add its binary directory to PATH.";
const SAVE_FAILED_MESSAGE =
  "AhdCode could not save the active file. The program was not run.";
const INVALID_EDITOR_MESSAGE =
  "Open a saved .ahd file before running AhdCode.";
const LSP_START_FAILED_MESSAGE =
  "AhdCode language server could not be started.\n" +
  "Diagnostics, hover, and other language features will be unavailable, but Run File still works.";

// The one argument `ahdcode lsp` accepts: start the stdio language server.
// This client requests no specific feature of its own -- vscode-languageclient
// wires up whatever the server's own `initialize` response advertises
// (diagnostics, hover, definition, document symbols, signature help,
// references, completion), so a server-side capability change never needs a
// matching client-side change here.
const LSP_ARGS = ["lsp"];

// TASK_TYPE matches the "type" this extension declares in package.json's
// contributes.taskDefinitions and the "type" runFile stamps on every Task
// it builds. Run File always constructs a fully-formed Task (definition,
// scope, and a concrete ProcessExecution) and hands it straight to
// tasks.executeTask -- plain VS Code accepts that without ever asking a
// TaskProvider for anything, since the task needs no resolving. Some
// VS Code-compatible hosts (Antigravity IDE, at least) validate the task's
// declared type against a *registered* provider before running it at all,
// independent of whether the task was already fully resolved, and refuse
// it with "there is no registered task type" otherwise. provideTasks
// returns none because Run File never asks the host to discover a task on
// its own (there is nothing to list in a "Run Task..." picker); this
// provider exists purely to satisfy that registration check.
const TASK_TYPE = "ahdcode";
const ahdcodeTaskProvider = {
  provideTasks() {
    return [];
  },
  resolveTask(task) {
    return task;
  },
};

// activate returns the promise VS Code awaits before considering the
// extension ready. Starting the language server is awaited here (rather
// than fired-and-forgotten) so a host or test never observes a half-started
// client, and so any startup failure is fully handled -- reported once via
// showErrorMessage -- before activate resolves.
async function activate(context, vscodeApi = vscode, runtime = {}, clientFactory = defaultClientFactory) {
  const disposable = vscodeApi.commands.registerCommand(COMMAND_ID, () => runFile(vscodeApi, runtime));
  context.subscriptions.push(disposable);
  context.subscriptions.push(vscodeApi.tasks.registerTaskProvider(TASK_TYPE, ahdcodeTaskProvider));
  await startLanguageClient(context, vscodeApi, runtime, clientFactory);
}

// languageClient is the one client this extension ever runs. startLanguageClient
// is idempotent -- called more than once (activate should only ever run once
// per extension host lifetime, but nothing prevents a caller from invoking it
// again) it starts nothing a second time.
let languageClient;

async function startLanguageClient(context, vscodeApi = vscode, runtime = {}, clientFactory = defaultClientFactory) {
  if (languageClient) {
    return;
  }
  const configuredPath = vscodeApi.workspace
    .getConfiguration("ahdcode")
    .get("executablePath", "")
    .trim();
  const executable = await findExecutable(
    configuredPath || "ahdcode",
    runtime.env || process.env,
    runtime.platform || process.platform,
  );
  if (!executable) {
    // Reported once, at activation, not on every hover or keystroke: Run
    // File already explains the same missing-executable problem when the
    // user tries to run something, so this is the same root cause surfaced
    // once up front rather than a second, repeated complaint.
    void vscodeApi.window.showErrorMessage(MISSING_EXECUTABLE_MESSAGE);
    return;
  }
  try {
    const client = clientFactory(
      { command: executable, args: LSP_ARGS },
      { documentSelector: [{ scheme: "file", language: "ahdcode" }] },
    );
    languageClient = client;
    context.subscriptions.push({ dispose: () => void stopLanguageClient() });
    await client.start();
  } catch (_error) {
    languageClient = undefined;
    void vscodeApi.window.showErrorMessage(LSP_START_FAILED_MESSAGE);
  }
}

// defaultClientFactory is the only place this file touches
// vscode-languageclient. serverOptions is the same {command, args} shape
// findExecutable/runFile already produce; run and debug are identical here
// since v0.2.0 has no separate debug transport.
function defaultClientFactory(serverOptions, clientOptions) {
  const { LanguageClient, TransportKind } = require("vscode-languageclient/node");
  const transported = { ...serverOptions, transport: TransportKind.stdio };
  return new LanguageClient(
    "ahdcode",
    "AhdCode Language Server",
    { run: transported, debug: transported },
    clientOptions,
  );
}

async function stopLanguageClient() {
  if (!languageClient) {
    return;
  }
  const client = languageClient;
  languageClient = undefined;
  await client.stop();
}

async function runFile(vscodeApi = vscode, runtime = {}) {
  const editor = vscodeApi.window.activeTextEditor;
  const document = editor && editor.document;

  if (!isRunnableDocument(document)) {
    void vscodeApi.window.showErrorMessage(INVALID_EDITOR_MESSAGE);
    return;
  }

  let saved;
  try {
    saved = await document.save();
  } catch (_error) {
    saved = false;
  }

  if (!saved || document.isDirty) {
    void vscodeApi.window.showErrorMessage(SAVE_FAILED_MESSAGE);
    return;
  }

  const configuredPath = vscodeApi.workspace
    .getConfiguration("ahdcode")
    .get("executablePath", "")
    .trim();
  const executable = await findExecutable(
    configuredPath || "ahdcode",
    runtime.env || process.env,
    runtime.platform || process.platform,
  );

  if (!executable) {
    void vscodeApi.window.showErrorMessage(MISSING_EXECUTABLE_MESSAGE);
    return;
  }

  const filePath = document.uri.fsPath;
  const workspaceFolder = vscodeApi.workspace.getWorkspaceFolder(document.uri);

  // A task terminal may only start in the user's home directory when no folder
  // is open, which would run a standalone file from the wrong directory or fail
  // outright. Such a file runs in its own terminal instead, still launched from
  // an argument vector rather than a shell command string.
  if (selectRunStrategy(workspaceFolder) === STANDALONE_STRATEGY) {
    runStandalone(vscodeApi, executable, filePath);
    return;
  }

  const taskName = `AhdCode: Run ${path.basename(filePath)}`;
  const execution = new vscodeApi.ProcessExecution(
    executable,
    ["run", filePath],
    { cwd: path.dirname(filePath) },
  );
  const scope = workspaceFolder || vscodeApi.TaskScope.Global;
  const task = new vscodeApi.Task(
    { type: TASK_TYPE, task: "runFile" },
    scope,
    taskName,
    "AhdCode",
    execution,
    [],
  );

  // One dedicated terminal is reused across runs, and it is never cleared, so
  // each run appends below the previous output instead of erasing it.
  task.presentationOptions = {
    reveal: vscodeApi.TaskRevealKind.Always,
    echo: true,
    focus: false,
    panel: vscodeApi.TaskPanelKind.Dedicated,
    showReuseMessage: false,
    clear: false,
  };
  task.runOptions = {
    instanceLimit: 1,
    reevaluateOnRerun: true,
  };

  try {
    await vscodeApi.tasks.executeTask(task);
  } catch (error) {
    const detail = error instanceof Error ? ` ${error.message}` : "";
    void vscodeApi.window.showErrorMessage(
      `AhdCode could not start the run task.${detail}`,
    );
  }
}

const TASK_STRATEGY = "task";
const STANDALONE_STRATEGY = "standalone";
const STANDALONE_TERMINAL_NAME = "AhdCode";

// selectRunStrategy decides how one run is launched. A document inside an open
// folder keeps the dedicated task terminal; a standalone file does not, because
// the host restricts a task terminal's working directory without a workspace.
function selectRunStrategy(workspaceFolder) {
  return workspaceFolder ? TASK_STRATEGY : STANDALONE_STRATEGY;
}

// standaloneTerminalOptions launches the compiler as the terminal's own
// process. shellPath and shellArgs are an argument vector, so a path containing
// spaces, quotes, $, ;, &, parentheses, or Unicode is never interpreted by a
// shell, and cwd stays the source file's directory so relative paths in the
// program still resolve the way they do on the command line.
function standaloneTerminalOptions(executable, filePath) {
  return {
    name: STANDALONE_TERMINAL_NAME,
    cwd: path.dirname(filePath),
    shellPath: executable,
    shellArgs: ["run", filePath],
  };
}

// standaloneTerminal is the one terminal this fallback owns. Its process ends
// when the program ends, so a rerun replaces it rather than reusing a dead
// shell; only one AhdCode terminal is ever left behind.
let standaloneTerminal;

function runStandalone(vscodeApi, executable, filePath) {
  if (standaloneTerminal) {
    standaloneTerminal.dispose();
    standaloneTerminal = undefined;
  }
  standaloneTerminal = vscodeApi.window.createTerminal(
    standaloneTerminalOptions(executable, filePath),
  );
  standaloneTerminal.show(true);
}

function isRunnableDocument(document) {
  return Boolean(
    document &&
      !document.isUntitled &&
      document.uri &&
      document.uri.scheme === "file" &&
      path.extname(document.uri.fsPath).toLowerCase() === ".ahd",
  );
}

async function findExecutable(command, environment, platform) {
  const value = command.trim();
  if (!value) {
    return undefined;
  }

  if (path.isAbsolute(value) || value.includes("/") || value.includes("\\")) {
    const candidate = path.resolve(value);
    return (await isExecutableFile(candidate, platform)) ? candidate : undefined;
  }

  const pathValue = environment.PATH || environment.Path || environment.path || "";
  const directories = pathValue.split(path.delimiter);
  const extensions = executableExtensions(value, environment, platform);

  for (const directory of directories) {
    if (!directory) {
      continue;
    }
    for (const extension of extensions) {
      const candidate = path.join(directory, value + extension);
      if (await isExecutableFile(candidate, platform)) {
        return candidate;
      }
    }
  }

  return undefined;
}

function executableExtensions(command, environment, platform) {
  if (platform !== "win32" || path.extname(command)) {
    return [""];
  }

  const pathExt = environment.PATHEXT || ".COM;.EXE;.BAT;.CMD";
  return pathExt
    .split(";")
    .filter(Boolean)
    .map((extension) => extension.toLowerCase());
}

async function isExecutableFile(candidate, platform) {
  try {
    const mode = platform === "win32" ? fs.constants.F_OK : fs.constants.X_OK;
    await fs.promises.access(candidate, mode);
    return (await fs.promises.stat(candidate)).isFile();
  } catch (_error) {
    return false;
  }
}

// deactivate returns stopLanguageClient()'s promise so VS Code awaits a
// clean server shutdown (the LSP shutdown/exit sequence) before the
// extension host tears the process down.
function deactivate() {
  return stopLanguageClient();
}

module.exports = {
  activate,
  selectRunStrategy,
  standaloneTerminalOptions,
  TASK_STRATEGY,
  STANDALONE_STRATEGY,
  deactivate,
  runFile,
  findExecutable,
  isRunnableDocument,
  startLanguageClient,
  stopLanguageClient,
  COMMAND_ID,
  INVALID_EDITOR_MESSAGE,
  MISSING_EXECUTABLE_MESSAGE,
  SAVE_FAILED_MESSAGE,
  LSP_START_FAILED_MESSAGE,
  LSP_ARGS,
};
