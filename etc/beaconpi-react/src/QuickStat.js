import React, { Component } from 'react';
import * as cfg from './config.js';
import { Table, Col, Row } from 'react-bootstrap';


class QuickStat extends Component {
  constructor(props, context) {
    super(props, context);
    this.state = {
      error: false,
      offlineEdges: [],
      offlineBeacons: [],
      beaconInactive: -1,
      edgeInactive: -1,
      beaconTotal: -1,
      edgeTotal: -1,
      hasData: false,
    }
  }

  componentDidMount() {
    var that = this;
    fetch(cfg.app + "/stats/quick", {
      method: 'GET',
      headers: {
        Accept: 'application/json',
      },
      credentials: 'include',
    }).then((r) => r.json())
    .then((rj) => {
      that.setState({
        offlineEdges: rj.InactiveEdges,
        offlineBeacons: rj.InactiveBeacons,
        beaconInactive: rj.InaBeaconCount,
        edgeInactive: rj.InaEdgeCount,
        beaconTotal: rj.BeaconCount,
        edgeTotal: rj.EdgeCount,
      });
    })
    .catch((error) => {
      that.setState({
        error: "Error occured fetching data from server"
      });
    });
  }

  render() {
    if (this.state.offlineEdges) {
      var iedge = this.state.offlineEdges.map((r) => {
        return (<tr key={r.Title}><td>{r.Title}</td><td>{r.Room}</td>
        <td>{r.Location}</td><td>{r.Description}</td></tr>);
      });
    }
    if (this.state.offlineBeacons) {
      var ibeacon = this.state.offlineBeacons.map((r) => {
        return (<tr key={r.Uuid}><td>{r.Label}</td><td>{r.Uuid}</td>
        <td>{r.Major}</td><td>{r.Minor}</td></tr>);
      });
    }
    if (this.state.hasData) {
      return (
        <h3>"Loading data!"</h3>
      )
    }
    return (
      <div>
      <Row>
        <Col md={12}>
          <h3>System Status</h3>
          <Table>
            <thead><tr>
            <th>Inactive Beacons</th>
            <th>Inactive Edges</th>
            </tr></thead>
            <tbody><tr>
            <td>{this.state.beaconInactive} / {this.state.beaconTotal}</td>
            <td>{this.state.edgeInactive} / {this.state.edgeTotal}</td>
            </tr></tbody>
          </Table>
        </Col>
      </Row>
      <Row>
        <Col md={12}>
          <h3>Inactive Edges</h3>
          <Table responsive>
            <thead><tr>
            <th>Title</th>
            <th>Room</th>
            <th>Location</th>
            <th>Description</th>
            </tr></thead>
            <tbody>
            {iedge}
            </tbody>
          </Table>
        </Col>
      </Row>
      <Row>
        <Col md={12}>
          <h3>Inactive Beacons</h3>
          <Table responsive>
            <thead><tr>
            <th>Label</th>
            <th>UUID</th>
            <th>Major</th>
            <th>Minor</th>
            </tr></thead>
            <tbody>
            {ibeacon}
            </tbody>
          </Table>
        </Col>
      </Row>
      </div>
    );
  }

}

export { QuickStat }
