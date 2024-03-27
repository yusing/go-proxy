function checkHealth(url, cell) {
  var xhttp = new XMLHttpRequest();
  xhttp.onreadystatechange = function () {
    if (this.readyState != 4) {
      return;
    }
    if (this.status === 200) {
      cell.innerHTML = '<div class="health-circle"></div>'; // Green circle for healthy
    } else {
      cell.innerHTML =
        '<div class="health-circle" style="background-color: #dc3545;"></div>'; // Red circle for unhealthy
    }
  };
  url =
    window.location.origin + "/checkhealth?target=" + encodeURIComponent(url);
  xhttp.open("HEAD", url, true);
  xhttp.send();
}

function updateHealthStatus() {
  let rows = document.querySelectorAll("tbody tr");
  rows.forEach((row) => {
    let url = row.querySelector("#url-cell").textContent;
    let cell = row.querySelector("#health-cell"); // Health column cell
    checkHealth(url, cell);
  });
}

document.addEventListener("DOMContentLoaded", () => {
  updateHealthStatus();

  // Update health status every 5 seconds
  setInterval(updateHealthStatus, 5000);
});
