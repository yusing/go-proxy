function contentIFrame() {
  return document.getElementById("content");
}

function openNavBtn() {
  return document.getElementById("openbtn");
}

function sideNav() {
  return document.getElementById("sidenav");
}

function setContent(path) {
  contentIFrame().attributes.src.value = path;
}

function openNav() {
  sideNav().style.width = "250px";
  contentIFrame().style.marginLeft = "250px";
  openNavBtn().style.display = "none";
}

function closeNav() {
  sideNav().style.width = "0";
  contentIFrame().style.marginLeft = "0px";
  openNavBtn().style.display = "inline-block";
}
