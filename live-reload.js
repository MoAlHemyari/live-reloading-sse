const evtSource = new EventSource("/updates");
let lastUpdate = "";

evtSource.onmessage = (e) => {
  if (lastUpdate === "") {
    lastUpdate = e.data;
    return;
  }
  if (e.data !== lastUpdate) {
    evtSource.close();
    lastUpdate = e.data;
    location.reload();
  }
};

let errorCount = 0;
evtSource.onerror = () => {
  errorCount++;
  if (errorCount >= 3) {
    console.error("Max errors reached. Stopping EventSource connection.");
    evtSource.close();
  }
};
