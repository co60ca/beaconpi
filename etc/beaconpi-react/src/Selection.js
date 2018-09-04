import React, { Component } from 'react';
import * as cfg from './config.js';

import { Label, FormGroup, FormControl, ControlLabel} from 'react-bootstrap';

// import Datetime from 'react-datetime';
// import 'react-datetime/css/react-datetime.css'

// Data should be loaded into
// id, description, data
// required props:
//   label: obvious
//   endpoint: string that contains the path on the webserver
//     "/api/test"
//   datatransform: function that takes a single json parameter
//     and returns a transform with id, description, data
//   idConsumer: Takes a list of id and returns nothing
//   errorConsumer: Takes an error message
// optional:
//   placeholder: text that is displayed as the first element
class MultiSelectLoad extends Component {
  constructor(props, context) {
    super(props, context);
    this.state = {
      data: []
    }
    this.onClick = this.onClick.bind(this);
    this.doFetch = this.doFetch.bind(this);

  }

  onClick(e) {
    var selected = Array.from(e.target.options)
      .filter(option => option.selected && option.index !== 0)
      .map(option => this.selectToOptions[option.value]);
      this.props.idConsumer(selected);
  }

  componentDidMount() {
    this.doFetch();
  }

  doFetch() {
    fetch(cfg.app + this.props.endpoint, {
      method: 'POST',
      headers: {
        Accept: 'application/json',
      },
      credentials: 'include',
    }).then((r) => r.json())
    .then((d) => {
      var o = this.props.datatransform(d);
      var i = 0;
      this.selectToOptions = new Array(this.state.data.length)
      for (let v of o) {
        this.selectToOptions[i++] = v.id;
      }
      this.setState({data: o});
      console.log(this.selectToOptions);
      console.log(o);
    })
    .catch((error) => {
      this.props.errorConsumer(error);
    });
  }

  render() {
    var i = 0;
    var eles = [
      <option key={0} value={null}>{this.props.placeholder || "Select an option..."}</option>
    ];
    this.state.data.forEach((v) => {
      eles.push(<option key={v.id} value={i++}>{v.description}</option>);
    });

    var multi = this.props.multi || false;

    return (
      <FormGroup controlId="formSelect" onChange={this.onClick}>
        <ControlLabel>{this.props.label}</ControlLabel>
        <FormControl componentClass="select" 
            placeholder="select" multiple={multi}
            style={{height: (this.props.height || "150") + "px"}}>
          {eles}
        </FormControl>
      </FormGroup>
    );
  }
}

export { MultiSelectLoad };
