"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const Module = require("node:module");
const os = require("node:os");
const path = require("node:path");
const test = require("node:test");

const state = {
  activeTextEditor: undefined,
  configurationPath: "",
  errors: [],
  tasks: [],
  handlers: new Map(),
};

class ProcessExecution {
  constructor(process, args, options) {
    this.process = process;
    this.args = args;
    this.options = options;
  }
}

class Task {
  constructor(definition, scope, name, source, execution, problemMatchers) {
    Object.assign(this, {
      definition,
      scope,
      name,
      source,
      execution,
      problemMatchers,
    });
  }
}

const vscodeMock = {
  commands: {
    registerCommand(command, handler) {
      state.handlers.set(command, handler);
      return { dispose() {} };
    },
  },
  window: {
    get activeTextEditor() {
      return state.activeTextEditor;
    },
    async showErrorMessage(message) {
      state.errors.push(message);
    },
  },
  workspace: {
    getConfiguration() {
      return {
        get(_name, defaultValue) {
          return state.configurationPath || defaultValue;
        },
      };
    },
    getWorkspaceFolder() {
      return undefined;
    },
  },
  tasks: {
    async executeTask(task) {
      state.tasks.push(task);
      return { task };
    },
  },
  ProcessExecution,
  Task,
  TaskScope: { Global: "global" },
  TaskRevealKind: { Always: "always" },
  TaskPanelKind: { Dedicated: "dedicated" },
};

const originalLoad = Module._load;
Module._load = function load(request, parent, isMain) {
  if (request === "vscode") {
    return vscodeMock;
  }
  return originalLoad.call(this, request, parent, isMain);
};
const extension = require("../extension");
Module._load = originalLoad;

function reset() {
  state.activeTextEditor = undefined;
  state.configurationPath = "";
  state.errors.length = 0;
  state.tasks.length = 0;
  state.handlers.clear();
}

function documentFor(filePath, options = {}) {
  return {
    isUntitled: false,
    isDirty: false,
    uri: { scheme: "file", fsPath: filePath },
    async save() {
      if (options.throwOnSave) {
        throw new Error("save failed");
      }
      return options.saveResult !== false;
    },
  };
}

function makeExecutable(directory) {
  const executable = path.join(directory, "ahdcode test executable");
  fs.writeFileSync(executable, "#!/bin/sh\nexit 0\n", { mode: 0o755 });
  return executable;
}

test.beforeEach(reset);

test("activate registers ahdcode.runFile", () => {
  const subscriptions = [];
  extension.activate({ subscriptions });
  assert.equal(subscriptions.length, 1);
  assert.equal(typeof state.handlers.get("ahdcode.runFile"), "function");
});

test("rejects a missing, untitled, or non-AhdCode editor", async () => {
  await extension.runFile(vscodeMock, { env: { PATH: "" } });
  assert.deepEqual(state.errors, [extension.INVALID_EDITOR_MESSAGE]);

  reset();
  const document = documentFor("/tmp/test.txt");
  state.activeTextEditor = { document };
  await extension.runFile(vscodeMock, { env: { PATH: "" } });
  assert.deepEqual(state.errors, [extension.INVALID_EDITOR_MESSAGE]);
  assert.equal(state.tasks.length, 0);
});

test("save failure prevents stale execution", async () => {
  const document = documentFor("/tmp/test.ahd", { saveResult: false });
  state.activeTextEditor = { document };

  await extension.runFile(vscodeMock, { env: { PATH: "" } });

  assert.deepEqual(state.errors, [extension.SAVE_FAILED_MESSAGE]);
  assert.equal(state.tasks.length, 0);
});

test("missing executable reports the required error without a cascade", async () => {
  state.activeTextEditor = { document: documentFor("/tmp/test.ahd") };

  await extension.runFile(vscodeMock, { env: { PATH: "" } });

  assert.deepEqual(state.errors, [extension.MISSING_EXECUTABLE_MESSAGE]);
  assert.equal(state.tasks.length, 0);
});

test("uses ProcessExecution argument arrays for spaces and Unicode paths", async () => {
  const temporary = fs.mkdtempSync(path.join(os.tmpdir(), "ahdcode-extension-"));
  const executable = makeExecutable(temporary);
  const filePath = "/tmp/Ahd Code Tests/öğrenci deneme.ahd";
  state.configurationPath = executable;
  state.activeTextEditor = { document: documentFor(filePath) };

  await extension.runFile(vscodeMock);

  assert.equal(state.errors.length, 0);
  assert.equal(state.tasks.length, 1);
  const task = state.tasks[0];
  assert.equal(task.name, "AhdCode: Run öğrenci deneme.ahd");
  assert.equal(task.execution.process, executable);
  assert.deepEqual(task.execution.args, ["run", filePath]);
  assert.deepEqual(task.execution.options, { cwd: "/tmp/Ahd Code Tests" });
  assert.equal(task.presentationOptions.panel, "dedicated");
  assert.equal(task.presentationOptions.clear, true);
  assert.equal(task.runOptions.instanceLimit, 1);

  fs.rmSync(temporary, { recursive: true, force: true });
});

test("findExecutable resolves ahdcode from PATH", async () => {
  const temporary = fs.mkdtempSync(path.join(os.tmpdir(), "ahdcode-path-"));
  const executable = path.join(temporary, "ahdcode");
  fs.writeFileSync(executable, "#!/bin/sh\nexit 0\n", { mode: 0o755 });

  assert.equal(
    await extension.findExecutable("ahdcode", { PATH: temporary }, process.platform),
    executable,
  );

  fs.rmSync(temporary, { recursive: true, force: true });
});

test("manifest exposes the portable command, menu, keybinding, and language", () => {
  const manifest = require("../package.json");
  assert.equal(manifest.main, "./extension.js");
  assert.equal(manifest.version, "0.1.1");
  assert.equal(manifest.engines.vscode, "^1.107.0");
  assert.equal(manifest.contributes.commands[0].command, "ahdcode.runFile");
  assert.equal(manifest.contributes.commands[0].icon, "$(play)");
  assert.equal(manifest.contributes.menus["editor/title"][0].when, "resourceLangId == ahdcode");
  assert.equal(manifest.contributes.keybindings[0].key, "f6");
  assert.equal(manifest.contributes.keybindings[0].when, "editorTextFocus && editorLangId == ahdcode");
  assert.deepEqual(manifest.contributes.languages[0].extensions, [".ahd"]);
});
