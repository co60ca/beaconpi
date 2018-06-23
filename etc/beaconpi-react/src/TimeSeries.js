import dateFormat from 'dateformat';
//import Fili from 'fili';
import React, { Component } from 'react';
import { Row, Col, FormGroup, FormControl,
  Alert, Table} from 'react-bootstrap';
import * as cfg from './config.js';
import { Line/*, defaults*/ } from 'react-chartjs-2';
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
      edges: [0, 1, 2],
      beacon: 1,
      edgelist: [],
      beaconlist: [],
      meandata: [],
      since: null,
      before: null,

      historysec: 2,
    }

    this.onEdgeSelect = this.onEdgeSelect.bind(this);
    this.onBeaconSelect = this.onBeaconSelect.bind(this);
  }



  onEdgeSelect(e) {
    var selected = Array.from(e.target.options)
      .filter(option => option.selected && option.index !== 0)
      .map(option => option.value);
    this.setState({
      edges: selected,
    });
  }

  onBeaconSelect(e) {
    this.setState({
      beacon: this.state.beaconlist[e.target.value].Id,
    });
  }

  updateSelection() {
    var that = this;
    fetch(cfg.app + "/config/alledges", {
      method: 'POST',
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

    fetch(cfg.app + "/config/allbeacons", {
      method: 'POST',
      headers: {
        Accept: 'application/json',
      },
      credentials: 'include',
    }).then((r) => r.json())
    .then((rj) => {
      that.setState({
        beaconlist: rj.Beacons
      })
    })
    .catch((error) => {
      that.setState({
        errortext: "Error receiving confirmation from server",
        message: "",
      });
    });
  }


  getData() {
    var before = new Date();
    before.setSeconds(before.getSeconds() - this.state.historysec);
    var after = new Date()
    after.setSeconds(after.getSeconds() - this.state.historysec - 1);
    before = dateFormat(before, 'isoUtcDateTime');
    after = dateFormat(after, 'isoUtcDateTime');


    var that = this;
    if (!this.state.edgelist) {
      return;
    }
    var edges = this.state.edges.map(e => this.state.edgelist[e].Id);
    fetch(cfg.app + "/history/short", {
      method: 'POST',
      headers: {
        Accept: 'application/json',
        'Content-Type': 'application/json',
      },
      credentials: 'include',
      body: JSON.stringify({
        Edges: edges,
        Beacon: that.state.beacon,
        Since: after,
        Before: before,
      }),
    }).then((r) => r.json())
    .then((rj) => {
      var idToChartIndex = new Map();
      var newdata = {labels: [], datasets: []};
      var scheme = new colorscheme();
      var colors = scheme.scheme('tetrade').variation('pastel').colors();
      for (var i in that.state.edges) {
        var thise = that.state.edgelist[that.state.edges[i]].Id;
        idToChartIndex.set(thise, i);
        newdata.datasets.push({
          label: 'Rssi ' + that.state.edgelist[that.state.edges[i]].Title + '',
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
        var first = that.state.edgelist[that.state.edges[0]].Id;
        if (e.Edge === first) {
          newdata.labels.push(e.Datetime);
        }
        
        var di = idToChartIndex.get(e.Edge);
        newdata.datasets[di].data.push(e.Rssi);
      }
      if (newdata.labels.length > 50) {
        for (i in newdata.datasets) {
          var d = newdata.datasets[i].data;
          while(d.length > 20) {
            d.shift();
          }
        }
        while(newdata.labels.length > 20) {
          newdata.labels.shift();
        }
      }
      var meandata = newdata.datasets.map(d => d.data.reduce((a, v) => a + v) / d.data.length);
      that.setState({
        chartdata: newdata,
        meandata: meandata,
        errortext: ""
      });
    })
    .catch((error) => {
      that.setState({
        errortext: "Error occured fetching data from server"
      });
    });
  }

  componentDidMount() {
    this.updateSelection();
    this.intervalGetData = setInterval(this.getData, 1000);
  }

  componentWillUnmount() {
    if (this.intervalGetData !== null) {
      clearInterval(this.intervalGetData);
    }
  }

  render() {
    var i = 0;
    var edgeEles = [
      <option key={0} value={null}>Select at least one edge...</option>
    ];
    this.state.edgelist.forEach((v) => {
      edgeEles.push(<option key={v.Id} value={i++}>{v.Title + 
        "\t" + v.Room + "\t" + v.Location}</option>);
    });

    var means = [];
    i = 0;
    if (this.state.meandata.length === this.state.edges.length) {
      this.state.edges.forEach(m => {
        means.push(<tr><td>{this.state.edgelist[m].Title}</td>
          <td>{this.state.meandata[i++]}</td></tr>);
      });
    }

    i = 0;
    var beaconEles = [
      <option key={0} value={null}>Select a beacon</option>
    ];
    this.state.beaconlist.forEach((v) => {
      beaconEles.push(<option key={v.Id} value={i++}>{v.Label}</option>);
    });

    return (
      <Row>
        <Col sm={12}>
          {this.state.message !== "" && 
            <Alert bsStyle="info">{this.state.message}</Alert>}
          {this.state.errortext !== "" && 
            <Alert bsStyle="danger">{this.state.errortext}</Alert>}
          <Line data={this.state.chartdata} 
              width={600} height={400} options={this.lineoptions} />
          <Table>
            <thead><tr><td>Edge</td><td>Average RSSI</td></tr></thead>
            <tbody>{means}</tbody>
          </Table>
          <form>
            <FormGroup controlId="formSelectEdge" onChange={this.onEdgeSelect}>
              <FormControl componentClass="select" 
                  placeholder="select" multiple
                  style={{height: "150px"}}>
                {edgeEles}
              </FormControl>
            </FormGroup>
            <FormGroup controlId="formSelectBeacon" 
                onChange={this.onBeaconSelect}>
              <FormControl componentClass="select" 
                  placeholder="select">
                {beaconEles}
              </FormControl>
            </FormGroup>
          </form>
        </Col>
      </Row>
    );
  }
}

export { Plot }
