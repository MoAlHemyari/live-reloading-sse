const evtSource = new EventSource("/updates");
const a = "hay"
let lastUpdate = "";

evtSource.onmessage = (event) => {
  if (lastUpdate === "") {
    lastUpdate = event.data;
    return;
  }
  if (event.data !== lastUpdate) {
    evtSource.close();
    lastUpdate = event.data;
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
