import React, { Component } from 'react';
//import * as cfg from './config.js';

import { Row, Col, Button, FormGroup, FormControl,
  Alert, ControlLabel } from 'react-bootstrap';

import Datetime from 'react-datetime';
import 'react-datetime/css/react-datetime.css'

import { MultiSelectLoad } from './Selection.js';


var edgeTransform = function(d) {
  return d.Edges.map(e => {
    return {
      id: e.Id,
      description: `${e.Title} - ${e.Room} - ${e.Location} - ${e.Description}`,
      data: e
    };
  });
};

var beaconTransform = function(d) {
  return d.Beacons.map(e => {
    return {
      id: e.Id,
      description: `${e.Label} - ${e.Uuid} - ${e.Major} - ${e.Minor}`,
      data: e
    };
  });
};

class Lateration extends Component {
  constructor(props, context) {
    super(props, context);
    this.state = {
      edgeList: [],
      beaconList: [],
      errortext: "",
      message: "",
      submitted: false,
    };
    this.handleError = this.handleError.bind(this);
    this.doSubmit = this.doSubmit.bind(this);
  }

  handleError(source, error) {
    console.log(source);
    console.log(error);
  }

  doSubmit() {
    
  }

  render() {

    return (
      <Row>
        <h4>Lateration</h4>
        <Col sm={12}>
          <form>
            <MultiSelectLoad label="Edges" endpoint="/config/alledges"
                datatransform={edgeTransform} 
                idConsumer={(ids) => {this.setState({edgeList: ids})}}
                errorConsumer={(error) => {this.handleError('edgelist', error)}}/>
            <MultiSelectLoad label="Beacons" endpoint="/config/allbeacons"
                datatransform={beaconTransform} 
                idConsumer={(ids) => {this.setState({beaconList: ids})}}
                errorConsumer={(error) => {this.handleError('beaconlist', error)}}/>
            <Button type="submit" 
            disabled={this.state.beaconList.length === 0 
            || this.state.edgeList === 0} onClick={this.doSubmit}>Display Filter</Button>
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

export { Lateration };
