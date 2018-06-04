import dateFormat from 'dateformat';
import Fili from 'fili';
import React, { Component } from 'react';
import * as cfg from './config.js';
import { Line, defaults } from 'react-chartjs-2';
import colorscheme from 'color-scheme';

class Plot extends Component {
//  constructor(props, context) {
//    super(props, context);
//  }

  render() {
    return <BeaconSeriesChart/>;
  }
}

function hexToRGB(hex, alpha) {
  var r = parseInt(hex.slice(1, 3), 16),
      g = parseInt(hex.slice(3, 5), 16),
      b = parseInt(hex.slice(5, 7), 16);

  return "rgba(" + r + ", " + g + ", " + b + ", " + alpha + ")";
}

class BeaconSeriesChart extends Component {
  constructor(props, context) {
    super(props, context);
    this.edges = [1, 2, 3];
    this.beacon = 1;
    this.chart = null;
    this.distance = false;
    this.intervalGetData = null;
    this.getData = this.getData.bind(this);


    this.lineoptions = {
      title: { text: 'Edge Node to Beacon Distance', display: true },
      scales: {
        yAxes: [{
          ticks: {
            suggestedMin: -100.0,
            suggestedMax: -30.0
          }
        }]
      }
    };


    this.state = {
      errortext: "",
      message: "",

      chartdata: {datasets:[], labels:[]},
      edgelist: [],
      since: null,
      before: null,

      historysec: 2,
    }
  }

  getData() {
    var before = new Date();
    before.setSeconds(before.getSeconds() - this.state.historysec);
    var after = new Date()
    after.setSeconds(after.getSeconds() - this.state.historysec - 1);
    before = dateFormat(before, 'isoUtcDateTime');
    after = dateFormat(after, 'isoUtcDateTime');


    var that = this;
    fetch(cfg.app + "/history/short", {
      method: 'POST',
      headers: {
        Accept: 'application/json',
        'Content-Type': 'application/json',
      },
      credentials: 'include',
      body: JSON.stringify({
        Edges: that.edges,
        Beacon: that.beacon,
        Since: after,
        Before: before,
      }),
    }).then((r) => r.json())
    .then((rj) => {
      var idToChartIndex = new Map();
      var newdata = {labels: [], datasets: []};
      var scheme = new colorscheme();
      var colors = scheme.scheme('tetrade').variation('pastel').colors();
      for (var i in that.edges) {
        idToChartIndex.set(that.edges[i], i);
        newdata.datasets.push({
          label: 'Rssi ' + that.edges[i] + '',
          data: [],
            backgroundColor: hexToRGB(colors[i%16], 0.2),
            borderColor: hexToRGB(colors[i%16], 1),
        });
      }

      for (i in that.state.chartdata.datasets) {
        var tds = that.state.chartdata.datasets[i];
        if (!newdata.datasets[i]) {
          continue;
        }
        var newds = newdata.datasets[i];
        if (tds.label !== newds.label) {
          continue;
        }
        newds.data.push.apply(newds.data, tds.data);
      }

      newdata.labels.push.apply(newdata.labels, that.state.chartdata.labels);

      for (i in rj) {
        var e = rj[i];
        var first = that.edges[i];
        if (e.Edge === first) {
          newdata.labels.push(e.Datetime);
        }
        
        var di = idToChartIndex.get(e.Edge);
        newdata.datasets[di].data.push(e.Rssi);
      }
      if (newdata.labels.length > 50) {
        for (i in newdata.datasets) {
          var d = newdata.datasets[i].data;
          while(d.length > 30) {
            d.shift();
          }
        }
        while(newdata.labels.length > 30) {
          newdata.labels.shift();
        }
      }
      that.setState({chartdata: newdata});
    })
    .catch((error) => {
      that.setState({
        error: "Error occured fetching data from server"
      });
    });
  }

  getEdgeList() {
    var that = this;
    fetch(cfg.app + "/config/alledges", {
      method: 'POST',
      headers: {
        Accept: 'application/json',
      },
      credentials: 'include',
    }).then((r) => r.json())
    .then((rj) => {
      that.setState({
        edgelist: rj.Edges
      })
    })
    .catch((error) => {
      that.setState({
        errortext: "Error receiving confirmation from server",
        message: "",
      });
    });
  }

  componentDidMount() {
    this.getEdgeList();
    this.intervalGetData = setInterval(this.getData, 1000);
  }

  componentWillUnmount() {
    if (this.intervalGetData !== null) {
      clearInterval(this.intervalGetData);
    }
  }

  render() {
    return <Line data={this.state.chartdata} 
    width={600} height={400} options={this.lineoptions} />
  }
}

export { Plot }
