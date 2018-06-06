import React, { Component } from 'react';
import * as cfg from './config.js';
import { FieldGroup } from './FormUtils.js';

import { Row, Col, Button, Checkbox, FormGroup, ControlLabel, FormControl,
  Alert } from 'react-bootstrap';

class AdminUserMod extends Component {
  constructor(props, context) {
    super(props, context);
    this.state = {
      userlist: [],
      errortext: "",
      message: "",
      id: false,
      email: "",
      password: "",
      dn: "",
      active: false,
      submitted: false,
    };
    this.handleChangeEmail = (e) => this.setState({email: e.target.value});
    this.handleChangePassword = (e) => this.setState({password: e.target.value});
    this.handleChangeDN = (e) => this.setState({dn: e.target.value});
    this.handleChangeActive = (e) => this.setState({active: e.target.value});
    
    this.doSubmit = this.doSubmit.bind(this);
    this.doNew = this.doNew.bind(this);
    this.doDelete = this.doDelete.bind(this);
    this.onUserSelect = this.onUserSelect.bind(this);
  }

  updateSelection() {
    var that = this;
    fetch(cfg.app + "/auth/allusers", {
      method: 'POST',
      headers: {
        Accept: 'application/json',
      },
      credentials: 'include',
    }).then((r) => r.json())
    .then((rj) => {
      that.setState({
        userlist: rj.Users
      })
    })
    .catch((error) => {
      that.setState({
        errortext: "Error receiving confirmation from server"
      });
    });
  }

  componentDidMount() {
    this.updateSelection();
  }

  sendRequest(obj, mod) {
    var that = this;
    fetch(cfg.app + "/auth/moduser", {
      method: 'POST',
      headers: {
        Accept: 'application/json',
        'Content-Type': 'application/json',
      },
      credentials: 'include',
      body: JSON.stringify(obj),
    }).then((r) => r.json())
    .then((rj) => {
      if ('Success' in rj) {
        that.setState({
          email: "",
          password: "",
          dn: "",
          active: false,
          submitted: false,
          message: "User " + mod + " complete",
          errortext: ""
        });
        this.updateSelection();
      }
    })
    .catch((error) => {
      that.setState({
        submitted: false,
        errortext: "Error receiving confirmation from server",
        message: "",
      });
    });
  }

  doSubmit(e) {
    // Mod
    this.sendRequest({
      "Id": this.state.id,
      "DisplayName": this.state.dn,
      "Email": this.state.email,
      "Passphrase": this.state.password === "<unchanged>" ? "" : 
          this.state.password,
      "Active": this.state.active,
      "Option": "mod",
    }, "modification");
  }
  doNew(e) {
    this.sendRequest({
      "DisplayName": this.state.dn,
      "Email": this.state.email,
      "Passphrase": this.state.password,
      "Active": this.state.active,
      "Option": "new",
    }, "add");
  }
  doDelete(e) {
    this.sendRequest({
      "Id": this.state.id,
      "Option": "rem",
    });
  }

  onUserSelect(e) {
    var s = this.state.userlist[e.target.value];
    if (!s) {
      return;
    }
    this.setState({
      id: s.Id,
      email: s.Email,
      password: "<unchanged>",
      dn: s.DisplayName,
      active: s.Active,
    });
  }

  render() {
    var userEles = [
      <option key={0} value={null}>Select Existing...</option>,
    ];
    var i = 0;
    this.state.userlist.forEach((v) => {
      userEles.push(<option key={v.Id} value={i++}>{v.DisplayName} ({v.Email})</option>);
    });
    return (
      <Row>
        <Col sm={12} md={6}>
          <form>
            <FormGroup controlId="formSelectUser">
              <FormControl componentClass="select" placeholder="select"
              onChange={this.onUserSelect}>
                {userEles}
              </FormControl>
            </FormGroup>
          </form>
          <form>
            <FieldGroup
              id="formControlDisplayName" type="text"
              label="Display Name" placeholder="Enter display name"
              value={this.state.dn}
              onChange={this.handleChangeDN}
              disabled={this.state.submitted}
            />
            <FieldGroup
              id="formControlEmail" type="email"
              label="Email address" placeholder="Enter email"
              value={this.state.email}
              onChange={this.handleChangeEmail}
              disabled={this.state.submitted}
            />
            <FieldGroup
              id="formControlPassword" type="password"
              label="Passphrase" placeholder="Enter passphrase"
              value={this.state.password}
              onFocus={() => this.state.password === '<unchanged>' && this.setState({password: ''}) }
              onChange={this.handleChangePassword}
              disabled={this.state.submitted}
            />
            <Checkbox title="Active title" checked={this.state.active} onChange={(e) => this.setState({active: e.target.checked})}>
              Active
            </Checkbox>
            <Button type="button" bsStyle="success" onClick={this.doNew}>New User</Button>
            {' '}
            <Button type="submit" disabled={!this.state.id} onClick={this.doSubmit}>Mod User</Button>
            {' '}
            <Button type="button" disabled={!this.state.id} bsStyle="danger" onClick={this.doDelete}>Delete User</Button>
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

class AdminModBeacon extends Component {
  constructor(props, context) {
    super(props, context);
    this.state = {
      beaconlist: [],
      errortext: "",
      id: false,
      label: "",
      uuid: "",
      major: "",
      minor: "",
      message: "",
      submitted: false,
    };

    this.handleChangeLabel = (e) => this.setState({label: e.target.value});
    this.handleChangeUuid = (e) => this.setState({uuid: e.target.value});
    this.handleChangeMajor = (e) => this.setState({major: parseInt(e.target.value, 10)});
    this.handleChangeMinor = (e) => this.setState({minor: parseInt(e.target.value, 10)});
    this.onBeaconSelect = this.onBeaconSelect.bind(this);
    this.doSubmit = this.doSubmit.bind(this);
    this.doNew = this.doNew.bind(this);
    this.doDelete = this.doDelete.bind(this);
  }

  updateSelection() {
    var that = this;
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
        submitted: false,
        errortext: "Error receiving confirmation from server",
        message: "",
      });
    });
  }

  onBeaconSelect(e) {
    var s = this.state.beaconlist[e.target.value];
    if (!s) {
      return;
    }
    this.setState({
      id: s.Id,
      label: s.Label,
      uuid: s.Uuid,
      major: s.Major,
      minor: s.Minor,
    });
  }

  componentDidMount() {
    this.updateSelection();
  }

  sendRequest(obj, mod) {
    var that = this;
    fetch(cfg.app + "/config/modbeacon", {
      method: 'POST',
      headers: {
        Accept: 'application/json',
        'Content-Type': 'application/json',
      },
      credentials: 'include',
      body: JSON.stringify(obj),
    }).then((r) => r.json())
    .then((rj) => {
      if ('Success' in rj) {
        that.setState({
          email: "",
          password: "",
          dn: "",
          active: false,
          submitted: false,
          message: "Beacon " + mod + " complete",
          errortext: ""
        });
        this.updateSelection();
      }
    })
    .catch((error) => {
      that.setState({
        submitted: false,
        errortext: "Error receiving confirmation from server",
        message: "",
      });
    });
  }

  doSubmit(e) {
    // Mod
    this.sendRequest({
      "Id": this.state.id,
      "Label": this.state.label,
      "Uuid": this.state.uuid,
      "Major": this.state.major,
      "Minor": this.state.minor,
      "Option": "mod",
    }, "modification");
  }
  doNew(e) {
    this.sendRequest({
      "Label": this.state.label,
      "Uuid": this.state.uuid,
      "Major": this.state.major,
      "Minor": this.state.minor,
      "Option": "new",
    }, "added");
  }
  doDelete(e) {
    this.sendRequest({
      "Id": this.state.id,
      "Option": "rem",
    }, "removals");
  }

  render() {
    var beaconEles = [
      <option key={0} value={null}>Select Existing...</option>,
    ];
    var i = 0;
    this.state.beaconlist.forEach((v) => {
      beaconEles.push(<option key={v.Id} value={i++}>{v.Label}</option>);
    });
    return (
      <Row>
        <Col sm={12} md={6}>
          <form>
            <FormGroup controlId="formSelectBeacon" onChange={this.onBeaconSelect}>
              <FormControl componentClass="select" placeholder="select">
                {beaconEles}
              </FormControl>
            </FormGroup>
          </form>
          <form>
            <FieldGroup
              id="formControlLabel" type="text"
              label="Display Name" placeholder="Enter Label"
              value={this.state.label}
              onChange={this.handleChangeLabel}
              disabled={this.state.submitted}
            />
            <FieldGroup
              id="formControlUuid" type="text"
              label="Uuid" placeholder="Enter Uuid (hex)"
              value={this.state.uuid}
              onChange={this.handleChangeUuid}
              disabled={this.state.submitted}
            />
            <FieldGroup
              id="formControlMajor" type="number"
              label="Major" placeholder="Enter Major"
              value={this.state.major}
              onChange={this.handleChangeMajor}
              disabled={this.state.submitted}
            />
            <FieldGroup
              id="formControlMinor" type="number"
              label="Minor" placeholder="Enter Minor"
              value={this.state.minor}
              onChange={this.handleChangeMinor}
              disabled={this.state.submitted}
            />

            <Button type="button" bsStyle="success" onClick={this.doNew}>New Beacon</Button>
            {' '}
            <Button type="submit" disabled={!this.state.id} onClick={this.doSubmit}>Mod Beacon</Button>
            {' '}
            <Button type="button" disabled={!this.state.id} bsStyle="danger" onClick={this.doDelete}>Delete Beacon</Button>
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

class AdminModEdge extends Component {
  constructor(props, context) {
    super(props, context);
    this.state = {
      edgeList: [],
      errortext: "",
      message: "",
      id: false,
      uuid: "",
      title: "",
      room: "",
      locationVal: "",
      description: "",
      bias: -50.0,
      gamma: 2.5,
      submitted: false,
    };

    this.onEdgeSelect = this.onEdgeSelect.bind(this);

    this.handleChangeUuid = (e) => this.setState({uuid: e.target.value});
    this.handleChangeTitle = (e) => this.setState({title: e.target.value});
    this.handleChangeRoom = (e) => this.setState({room: e.target.value});
    this.handleChangeLocation = (e) => this.setState({locationVal: e.target.value});
    this.handleChangeDescription = (e) => this.setState({description: e.target.value});
    this.handleChangeBias = (e) => this.setState({bias: parseFloat(e.target.value)});
    this.handleChangeGamma = (e) => this.setState({gamma: parseFloat(e.target.value)});

    this.doSubmit = this.doSubmit.bind(this);
    this.doNew = this.doNew.bind(this);
    this.doDelete = this.doDelete.bind(this);
  }

  componentDidMount() {
    this.updateSelection();
  }

  sendRequest(obj, mod) {
    var that = this;
    fetch(cfg.app + "/config/modedge", {
      method: 'POST',
      headers: {
        Accept: 'application/json',
        'Content-Type': 'application/json',
      },
      credentials: 'include',
      body: JSON.stringify(obj),
    }).then((r) => r.json())
    .then((rj) => {
      if ('Success' in rj) {
        that.setState({
          errortext: "",
          id: false,
          uuid: "",
          title: "",
          room: "",
          locationVal: "",
          description: "",
          bias: -50.0,
          gamma: 2.5,
          submitted: false,
        });
        this.updateSelection();
      }
    })
    .catch((error) => {
      that.setState({
        submitted: false,
        errortext: "Error receiving confirmation from server",
        message: "",
      });
    });
  }

  onEdgeSelect(e) {
    var s = this.state.edgeList[e.target.value];
    if (!s) {
      return;
    }
    this.setState({
      id: s.Id,
      uuid: s.Uuid,
      title: s.Title,
      room: s.Room,
      locationVal: s.Location,
      description: s.Description,
      bias: s.Bias,
      gamma: s.Gamma,
    });
  }

  updateSelection() {
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
  }

  doSubmit(e) {
    this.sendRequest({
      "Id": this.state.id,
      "Uuid": this.state.uuid,
      "Title": this.state.title,
      "Room": this.state.room,
      "Location": this.state.locationVal,
      "Description": this.state.description,
      "Bias": this.state.bias,
      "Gamma": this.state.gamma,
      "Option": "mod",
    }, "modification");
  }
  doNew(e) {
    this.sendRequest({
      "Uuid": this.state.uuid,
      "Title": this.state.title,
      "Room": this.state.room,
      "Location": this.state.locationVal,
      "Description": this.state.description,
      "Bias": this.state.bias,
      "Gamma": this.state.gamma,
      "Option": "new",
    }, "added");
  }
  doDelete(e) {
    this.sendRequest({
      "Id": this.state.id,
      "Option": "rem",
    }, "removed");
  }

  render() {
    var edgeEles = [
      <option key={0} value={null}>Select Existing...</option>,
    ];
    var i = 0;
    this.state.edgeList.forEach((v) => {
      edgeEles.push(<option key={v.Id} value={i++}>{v.Title + 
        "\t" + v.Room + "\t" + v.Location}</option>);
    });
    return (
      <Row>
        <Col sm={12} md={6}>
          <form>
            <FormGroup controlId="formSelectEdge" onChange={this.onEdgeSelect}>
              <FormControl componentClass="select" placeholder="select">
                {edgeEles}
              </FormControl>
            </FormGroup>
          </form>
          <form>
            <FieldGroup
              id="formControlUuid" type="text"
              label="Uuid" placeholder="Enter Uuid (hex)"
              value={this.state.uuid}
              onChange={this.handleChangeUuid}
              disabled={this.state.submitted}
            />
            <FieldGroup
              id="formControlTitle" type="text"
              label="Title" placeholder="Enter Title"
              value={this.state.title}
              onChange={this.handleChangeTitle}
              disabled={this.state.submitted}
            />
            <FieldGroup
              id="formControlRoom" type="text"
              label="Room" placeholder="Enter Room"
              value={this.state.room}
              onChange={this.handleChangeRoom}
              disabled={this.state.submitted}
            />
            <FormGroup controlId="formControlLocation">
              <ControlLabel>Location</ControlLabel>
              <FormControl componentClass="textarea" placeholder="Enter Location"
              value={this.state.locationVal}
              onChange={this.handleChangeLocation}
              disabled={this.state.submitted}/>
            </FormGroup>
            <FormGroup controlId="formControlDescription">
              <ControlLabel>Description</ControlLabel>
              <FormControl componentClass="textarea" placeholder="Enter Description"
              value={this.state.description}
              onChange={this.handleChangeDescription}
              disabled={this.state.submitted}/>
            </FormGroup>
            <FieldGroup
              id="formControlBias" type="number"
              label="Bias" placeholder="Enter bias"
              value={this.state.bias}
              onChange={this.handleChangeBias}
              disabled={this.state.submitted}
            />
            <FieldGroup
              id="formControlGamma" type="number"
              label="Gamma" placeholder="Enter gamma"
              value={this.state.gamma}
              onChange={this.handleChangeGamma}
              disabled={this.state.submitted}
            />

            <Button type="button" bsStyle="success" onClick={this.doNew}>New Edge</Button>
            {' '}
            <Button type="submit" disabled={!this.state.id} onClick={this.doSubmit}>Mod Edge</Button>
            {' '}
            <Button type="button" disabled={!this.state.id} bsStyle="danger" onClick={this.doDelete}>Delete Edge</Button>
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

export { AdminUserMod, AdminModBeacon, AdminModEdge };
