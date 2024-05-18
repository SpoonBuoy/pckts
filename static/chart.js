// chart.js

// Initialize Chart.js
const ctx = document.getElementById('packetRateChart').getContext('2d');

const data = {
    labels: [], // Time labels
    datasets: [
        {
            label: 'Client 1',
            data: [],
            borderColor: 'red',
            fill: false
        },
        {
            label: 'Client 2',
            data: [],
            borderColor: 'blue',
            fill: false
        },
        {
            label: 'Client 3',
            data: [],
            borderColor: 'green',
            fill: false
        },
        {
            label: 'Server',
            data: [],
            borderColor: 'black',
            fill: false
        }
    ]
};

const config = {
    type: 'line',
    data: data,
    options: {
        scales: {
            x: {
                type: 'realtime',
                realtime: {
                    duration: 20000, // 20 seconds of data
                    refresh: 1000, // refresh every second
                    delay: 1000, // delay of 1 second
                    onRefresh: chart => {
                        // This function is called to update the chart
                    }
                }
            },
            y: {
                beginAtZero: true
            }
        }
    }
};

const packetRateChart = new Chart(ctx, config);

// Function to update chart with new data
function updateChart(data) {
    const now = Date.now();
    
    packetRateChart.data.datasets[0].data.push({ x: now, y: data.client1.Rate });
    packetRateChart.data.datasets[1].data.push({ x: now, y: data.client2.Rate });
    packetRateChart.data.datasets[2].data.push({ x: now, y: data.client3.Rate });
    packetRateChart.data.datasets[3].data.push({ x: now, y: data.server.TotalPacketRate });
    
    packetRateChart.update('quiet');
}

// WebSocket for receiving data
rcvSocket.onmessage = function(event) {
    const data = JSON.parse(event.data);    
    document.getElementById('client1-rate').innerText = data.client1.Rate;
    document.getElementById('client2-rate').innerText = data.client2.Rate;
    document.getElementById('client3-rate').innerText = data.client3.Rate;
    document.getElementById('server-rate').innerText = data.server.TotalPacketRate;
            
    document.getElementById('client1-totPackets').innerText = data.client1.TotalPacketsReceived;
    document.getElementById('client2-totPackets').innerText = data.client2.TotalPacketsReceived;
    document.getElementById('client3-totPackets').innerText = data.client3.TotalPacketsReceived;
    document.getElementById('server-totPackets').innerText = data.server.TotalPacketsReceived;
          
    updateChart(data);

};
