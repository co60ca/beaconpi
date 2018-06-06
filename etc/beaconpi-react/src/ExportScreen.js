import React, { Component } from 'react';
import * as cfg from './config.js';

import { Row, Col, Button, FormGroup, FormControl,
  Alert, ControlLabel } from 'react-bootstrap';

import Datetime from 'react-datetime';
import 'react-datetime/css/react-datetime.css'

class ExportData extends Component {
  constructor(props, context) {
    super(props, context);
    this.state = {
      edgeList: [],
      beaconList: [],
      selectedEdge: [],
      selectedBeacon: [],
      before: false,
      after: false,
      errortext: "",
      message: "",
      submitted: false,
    };

    this.onEdgeSelect = this.onEdgeSelect.bind(this);
    this.onBeaconSelect = this.onBeaconSelect.bind(this);
    this.doSubmit = this.doSubmit.bind(this);
    this.changeAfter = this.changeAfter.bind(this);
    this.changeBefore = this.changeBefore.bind(this);
  }

  doSubmit() {
    var before = this.state.before;
    var after = this.state.after;

    if (!before || !after) {
      this.setState({errortext: "Set both After and Before"});
      return;
    }

    if (after.isAfter(before)) {
      this.setState({errortext: "After must be before before"});
      return;
    }
    this.sendRequest({
      "Beacons": this.state.selectedBeacon.map(b => b.Id),
      "Edges": this.state.selectedEdge.map(e => e.Id),
      "After": this.state.after,
      "Before": this.state.before,
    });
  }

  changeAfter(e) {
    //this.after = e.format('YYYY-MM-DD[T]HH:mm:ss.SSSSSSSSSZ');
    this.setState({
      after: e,
    });
  }

  changeBefore(e) {
    //this.before = e.format('YYYY-MM-DD[T]HH:mm:ss.SSSSSSSSSZ');
    this.setState({
      before: e,
    });
  }

  componentDidMount() {
    this.updateSelection();
  }

  updateSelection() {
    var that = this;
    fetch(cfg.app + "/config/alledges", {
      method: 'POST',
      credentials: 'include',
    }).then((r) => r.json())
    .then((rj) => {
      that.setState({
        edgeList: rj.Edges
      })
    })
    .catch((error) => {
      that.setState({
        submitted: false,
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
        beaconList: rj.Beacons
      })
    })
    .catch((error) => {
      that.setState({
        submitted: false,
        errortext: "Error receiving confirmation from server",
        message: "",
      });
    });
  }

  sendRequest(obj) {
    var that = this;
    fetch(cfg.app + "/history/export", {
      method: 'POST',
      headers: {
        Accept: 'application/json',
        'Content-Type': 'application/json',
      },
      credentials: 'include',
      body: JSON.stringify(obj),
    })
    .then(response => response.blob())
    .then(blob => URL.createObjectURL(blob, {type: 'octet/stream'}))
    .then(url => {
      // Gross hack
      var a = document.createElement("a");
      document.body.append(a);
      a.style = "display: none";
      a.href = url;
      a.download = "" + (new Date()).toISOString().substr(0, 19) + "Z_beaconpi_export.csv";
      a.click();
      URL.revokeObjectURL(url);
      that.setState({
        submitted: false,
        errortext: "",
        message: "",
      })
    })
    .catch((error) => {
      that.setState({
        submitted: false,
        errortext: "Error receiving export data from server",
        message: "",
      });
    });

    this.setState({
      submitted: true,
    })
  }

  onEdgeSelect(e) {
    var selected = Array.from(e.target.options)
      .filter(option => option.selected && option.index !== 0)
      .map(option => option.value)
      .map(e => this.state.edgeList[e]);
    this.setState({
      selectedEdge: selected,
    });
  }

  onBeaconSelect(e) {
    var selected = Array.from(e.target.options)
      .filter(option => option.selected && option.index !== 0)
      .map(option => option.value)
      .map(e => this.state.beaconList[e]);
    this.setState({
      selectedBeacon: selected,
    });
  }
  

  render() {
    var i = 0;
    var edgeEles = [
      <option key={0} value={null}>Select at least one edge</option>
    ];
    this.state.edgeList.forEach((v) => {
      edgeEles.push(<option key={v.Id} value={i++}>{v.Title}</option>);
    });

    i = 0;
    var beaconEles = [
      <option key={0} value={null}>Select at least one beacon</option>
    ];
    this.state.beaconList.forEach((v) => {
      beaconEles.push(<option key={v.Id} value={i++}>{v.Label}</option>);
    });

    return (
      <Row>
        <Col sm={12}>
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
                  placeholder="select" multiple
                  style={{height: "150px"}}>
                {beaconEles}
              </FormControl>
            </FormGroup>
            <Row>
              <Col sm={12} md={6}>
                <ControlLabel>After</ControlLabel>
                <DTPicker onChange={this.changeAfter}/>
              </Col>
              <Col sm={12} md={6}>
                <ControlLabel>Before</ControlLabel>
                <DTPicker onChange={this.changeBefore}/>
              </Col>
            </Row>
            <Button type="submit" disabled={
              this.state.selectedEdge.length === 0 
              || this.state.selectedBeacon.length === 0
              || !this.state.before
              || !this.state.after
              || this.state.submitted}
              onClick={this.doSubmit}>{this.state.submitted ? 
                'Waiting...' : 'Request CSV'}</Button>
          </form>
          {this.state.message !== "" && 
            <Alert bsStyle="info">{this.state.message}</Alert>}
          {this.state.errortext !== "" && 
            <Alert bsStyle="danger">{this.state.errortext}</Alert>}
        </Col>
      </Row>
    );
  }
}

class DTPicker extends Component {
    render() {
        return (
          <div>
          <Datetime {...this.props} input={false}/>
          </div>
        )
    }
}

export default ExportData;
