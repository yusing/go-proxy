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

function loadFile(fileName) {
  if (fileName === undefined) {
    return;
  }
  let req = new XMLHttpRequest();
  req.open("GET", `/config/${fileName}`, true);
  req.onreadystatechange = function () {
    if (req.readyState == 4) {
      if (req.status == 200) {
        let old_nav_item = document.getElementById(`file-${currentFile}`);
        old_nav_item.classList.remove("active");
        editor.setValue(req.responseText);
        currentFile = fileName;
        let new_nav_item = document.getElementById(`file-${currentFile}`);
        new_nav_item.classList.add("active");
        document.title = `${currentFile} - Config Editor`;
        console.log(`loaded ${currentFile}`);
      } else {
        let msg = `Failed to load ${fileName}: ` + req.responseText;
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
        alert("Saved " + filename);
      } else {
        alert("Error: " + req.responseText);
      }
    }
  };
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