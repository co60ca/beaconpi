import React, { Component } from 'react';
import * as cfg from './config.js';

import { Row, Col, Button, FormGroup, FormControl,
  Alert, ControlLabel } from 'react-bootstrap';
import { Stage, Layer, Star, Text, Image } from 'react-konva';
import Datetime from 'react-datetime';
import 'react-datetime/css/react-datetime.css'

import { MultiSelectLoad } from './Selection.js';

const TIMEOUT_MS_STEPS = 500;

var mapTransform = function(d) {
  return d.Maps.map(e => {
    return {
      id: e.Id,
      description: `${e.Title}`,
      data: e
    };
  });
};

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

class Map extends Component {
  constructor(props, context) {
    super(props, context);
    this.state = {};
  }

  componentDidMount() {
    var that = this;
    fetch(cfg.app + '/maps/mapimage', {
      method: 'POST',
      headers: {
        Accept: 'application/json',
        'Content-Type': 'application/json',
      },
      credentials: 'include',
      body: JSON.stringify({'ImageID': this.props.resource}),
    })
    .then(response => response.blob())
    .then(blob => URL.createObjectURL(blob))
    .then(url => {
      var img = new window.Image();
      img.src = url;
      that.setState({image: img});
    })
    .catch(that.errorConsumer);
  }

  render() {
    return <Image image={this.state.image} />;
  }
}

class Lateration extends Component {
  constructor(props, context) {
    super(props, context);
    this.state = {
      edgeList: [],
      beaconList: [],
      map: null,
      errortext: "",
      message: "",
      submitted: false,
      // Testing
      formopen: false,
      //formopen: true,
      mapimg: 49454
    };
    this.konva_beacons = [];
    this.handleError = this.handleError.bind(this);
    this.doSubmit = this.doSubmit.bind(this);
  }

  handleError(source, error) {
    console.log(`source: ${source}, error: ${error}`);
  }

  startLoop() {
    if (!this.looptimeout) {
      this.looptimeout = setTimeout(this.loop, TIMEOUT_MS_STEPS);
    }
  }

  updateDisplay() {
    //TODO(mae)

  }

  loop() {
    var that = this;
    fetch(cfg.app + '/history/maptracking', {
      method: 'POST',
      headers: {
        Accept: 'application/json',
        'Content-Type': 'application/json',
      },
      credentials: 'include',
      body: JSON.stringify(this.request),
    }).then((r) => r.json())
    .then((d) => {
      // Update the filterID if we were reset by the server
      try {
        that.request.FilterID = d.FilterID;
        that.setState({
          edgeList: d.Edges,
          beaconList: d.Beacons,
          mapConfig: d.MapConfig,
          timeData: d.Series
        });
        that.updateDisplay();
      } finally {
        that.startLoop();
      }
    })
    .catch((error) => {
      that.handleError('maptracking', error);
    });
  }

  doSubmit() {
    
  }

  render() {

    return (
      <Row>
        <h4>Lateration</h4>
        <Col sm={12}>
        {!this.state.formopen && 
          <Stage>
            <Layer>
              <Map resource={this.state.mapimg} errConsumer={(e) => this.handleError('img', e)}/>
            </Layer> 
            <Layer>
              {this.beacons}
            </Layer>
          </Stage>
        }
        {this.state.formopen &&
          <form>
            <MultiSelectLoad label="Map" endpoint="/maps/allmaps"
                datatransform={mapTransform} 
                idConsumer={(ids) => {this.setState({map: ids})}}
                errorConsumer={(error) => {this.handleError('maplist', error)}}
                height='50px'/>
            <MultiSelectLoad label="Edges" endpoint="/config/alledges"
                datatransform={edgeTransform} multi={true}
                idConsumer={(ids) => {this.setState({edgeList: ids})}}
                errorConsumer={(error) => {this.handleError('edgelist', error)}}/>
            <MultiSelectLoad label="Beacons" endpoint="/config/allbeacons"
                datatransform={beaconTransform} multi={true}
                idConsumer={(ids) => {this.setState({beaconList: ids})}}
                errorConsumer={(error) => {this.handleError('beaconlist', error)}}/>
            <Button type="submit" 
            disabled={this.state.beaconList.length === 0 
            || this.state.edgeList === 0} onClick={this.doSubmit}>Display Filter</Button>
          </form>
        }
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
