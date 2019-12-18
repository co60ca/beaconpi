import React, { Component } from 'react';
import * as cfg from './config.js';

import { Row, Col, Button, Alert} from 'react-bootstrap';
import { Circle, Stage, Layer, Image } from 'react-konva';
import 'react-datetime/css/react-datetime.css'
import dateFormat from 'dateformat';

import { MultiSelectLoad } from './Selection.js';

const TIMEOUT_MS_STEPS = 100;
const CIRCLE_BASESIZE = 40;

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
      img.onload = () => {
        that.setState({image: img});
      }
    })
    .catch(that.errorConsumer);
  }

  render() {
    return <Image image={this.state.image} scale={this.props.scale} />;
  }
}

class Lateration extends Component {
  constructor(props, context) {
    super(props, context);
    this.state = {
      edgeList: [],
      beaconList: [],
      map: 1,
      errortext: "",
      message: "",
      submitted: false,
      formopen: true,
      stagewidth: 1000,
      stageheight: 1000,
      uiscale: {x: 0.5, y: 0.5},
      mapscale: {x: 1, y: 1},
      offset: {x: 0.0, y: 0.0}
    };
    this.handleError = this.handleError.bind(this);
    this.doSubmit = this.doSubmit.bind(this);
    this.updateBeacons = this.updateBeacons.bind(this);
    this.loop = this.loop.bind(this);

    this.uibeacons = [];
    this.uibeaconref = [];
    this.uibeaconloc = [];
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

    var mc = this.state.mapConfig;
    this.setState({
      offset: {x: mc.CoordBiasX, y:mc.CoordBiasY},
      mapscale: {x: mc.CoordScaleX, y: mc.CoordScaleY}
    });/*
    for (let b of this.uibeaconref) {
      b.scale(this.state.uiscale);
    }
    for (let t in this.state.timeData) {
      // TODO always sort results?
      var ts = this.state.timeData[t];
      var ele = this.state.beaconList.indexOf(ts.Beacon);
      if (ele === -1) {
        throw "Beacon not in list!";
      }
      this.uibeaconlog[ele] = loc;
    }
*/
  }
  
  updateBeacons(newbeacons) {
  }

  loop() {
    this.looptimeout = null;
    var that = this;

    this.request.RequestTime = dateFormat(new Date(Date.now()-3000), 'isoUtcDateTime');
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
        var loc = new Array(d.Series.length);
        for (let i in d.Series) {
          let index = d.Beacons.indexOf(d.Series[i].Beacon);
          if (index === -1) {
            throw "Unfound index";
          }
          let ts = d.Series[i];
          let tloc = {
            x: this.state.offset.x * this.state.uiscale.x + this.state.mapscale.x * ts.Location[0] * this.state.uiscale.x,
            y: this.state.offset.y * this.state.uiscale.y + this.state.mapscale.y * ts.Location[1] * this.state.uiscale.y
          }
          loc[index] = tloc;
          
        }
        that.setState({
          edgeList: d.Edges,
          beaconList: d.Beacons,
          mapConfig: d.MapConfig,
          timeData: d.Series,
          formopen: false,
          uibeaconsloc: loc
        });
        that.updateDisplay();
      } finally {
        that.startLoop();
      }
    })
    .catch((error) => {
      that.handleError('maptracking', error);
      that.startLoop();
    });
  }

  doSubmit() {
    this.request = {
      FilterID: "",
      Beacons: this.state.beaconList,
      Edges: this.state.edgeList,
      MapID: this.state.map,
      RequestTime: dateFormat(new Date(Date.now()-3000), 'isoUtcDateTime'),
      Algorithm: "particle-filter-velocity"
    };
    this.loop();    
  }

  render() {
    let beacons = [];
    if (this.state.uibeaconsloc) {
      for (let i in this.state.uibeaconsloc) {
        var curloc = this.state.uibeaconsloc[i];

        var circle = <Circle radius={CIRCLE_BASESIZE} fill={'red'} stroke={'black'}
          strokeWidth={2} {...curloc} scale={this.state.uiscale}/>

        beacons.push(circle);
      }
    }
    return (
      <Row>
        <h4>Lateration</h4>
        <Col sm={12}>
          {!this.state.formopen && 
            <Stage height={this.state.stageheight} width={this.state.stagewidth} 
                draggable>
              <Layer>
                <Map ref={r => {this.elemap = r}} resource={this.state.map} 
                  errConsumer={(e) => this.handleError('img', e)} 
                  scale={this.state.uiscale}
                  />
              </Layer> 
              <Layer>
                {beacons}
              </Layer>
            </Stage>

          }
          {this.state.formopen &&
            <form>
              <MultiSelectLoad label="Map" endpoint="/maps/allmaps"
                  datatransform={mapTransform} 
                  idConsumer={(ids) => {this.setState({map: (ids && ids.length === 1) ? ids[0] : null})}}
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
              <Button type="button" 
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
