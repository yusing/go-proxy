let currentFile = "config.yml";
let editorElement = document.getElementById("config-editor");
let fileListElement = document.getElementById("file-list");
let editor = CodeMirror(editorElement, {
  lineNumbers: true,
  mode: "yaml",
  theme: "dracula",
  autofocus: true,
  lineWiseCopyCut: true,
  keyMap: "sublime",
  tabSize: 2
});

function setCurrentFile(filename) {
  let old_nav_item = document.getElementById(`file-${currentFile}`);
  if (old_nav_item !== null) {
    old_nav_item.classList.remove("active");
  }
  currentFile = filename;
  document.title = `${currentFile} - Config Editor`;
  let new_nav_item = document.getElementById(`file-${currentFile}`);
  if (new_nav_item === null) {
    new_file_btn = document.getElementById("new-file");
    file_list = document.getElementById("file-list");
    new_nav_item = document.createElement("li");
    new_nav_item.id = `file-${currentFile}`;
    new_nav_item.innerHTML = `<a class="unselectable">${currentFile}</a>`;
    file_list.insertBefore(new_nav_item, new_file_btn);
  }
  new_nav_item.classList.add("active");
}

function loadFile(filename) {
  if (filename === undefined) {
    return;
  }
  if (filename === '+') {
    newFile();
    return;
  }
  let req = new XMLHttpRequest();
  req.open("GET", `/config/${filename}`, true);
  req.onreadystatechange = function () {
    if (req.readyState == 4) {
      if (req.status == 200) {
        editor.setValue(req.responseText);
        setCurrentFile(filename);
        console.log(`loaded ${currentFile}`);
      } else {
        let msg = `Failed to load ${filename}: ` + req.responseText;
        alert(msg);
        console.log(msg);
      }
    }
  };
  req.send();
}

function saveFile(filename, content) {
  let req = new XMLHttpRequest();
  req.open("PUT", `/config/${filename}`, true);
  req.setRequestHeader("Content-Type", "text/plain");
  req.send(content);
  req.onreadystatechange = function () {
    if (req.readyState == 4) {
      if (req.status == 200) {
        alert(req.responseText);
      } else {
        alert("Error:\n" + req.responseText);
      }
    }
  };
}

function newFile() {
  let filename = prompt("Enter filename:");
  if (filename === undefined || filename === "") {
    alert("File name cannot be empty");
    return;
  }
  if (!filename.endsWith(".yml") && !filename.endsWith(".yaml")) {
    alert("File name must end with .yml or .yaml");
    return;
  }
  let files = document.getElementById("file-list").children;
  for (let i = 0; i < files.length; i++) {
    if (files[i].id === `file-${filename}`) {
      alert("File already exists");
      return;
    }
  }
  editor.setValue("");
  setCurrentFile(filename);
}

editor.setSize("100wh", "100vh");
editor.setOption("extraKeys", {
  Tab: function (cm) {
    const spaces = Array(cm.getOption("indentUnit") + 1).join(" ");
    cm.replaceSelection(spaces);
  },
  "Ctrl-S": function (cm) {
    saveFile(currentFile, cm.getValue());
  },
});
fileListElement.addEventListener("click", function (e) {
    if (e.target === null) {
        return;
    }
    loadFile(e.target.text);
});
function onLoad() {
  loadFile(currentFile);
}