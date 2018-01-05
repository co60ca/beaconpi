var Fili = require('fili');
var dateFormat = require('dateformat');
var trilat = require('trilat');
var Chart = require('chart.js');

var target = 'http://3508data.soe.uoguelph.ca:32967'

/*
function getTrilateration(position1, position2, position3) {
  var input = [
  //      X     Y     R 
  [position1.x, position1.y, position1.distance],
  [position2.x, position2.y, position2.distance],
  [position3.x, position3.y, position3.distance]];
  console.log(input);
  var output = trilat(input);
  return { x: output[0], y: output[1] };
}
*/
function getTrilateration(positions) {
  positions.forEach((p) => {
    p.distance = p.distance || Number.POSITIVE_INFINITY;
  });
  var midpointx = positions.reduce(function(a, b) {
    return Math.max(a.x || a, b.x);
  })/2 + positions.reduce(function(a, b) {
    return Math.min(a.x || a, b.x);
  })/2;
  var midpointy = positions.reduce(function(a, b) {
    return Math.max(a.y || a, b.y);
  })/2 + positions.reduce(function(a, b) {
    return Math.min(a.y || a, b.y);
  })/2

  var closest = 0;
  var closestval = positions[0].distance;
  for (var i in positions) {
    if (closestval > positions[i].distance) {
      closestval = positions[i].distance;
      closest = i;
    }
  }

  var x = (3*positions[closest].x + midpointx) / 4 
  var y = (3*positions[closest].y + midpointy) / 4 
  return { x: x, y: y };
}

var dps = [];
var dbl = [];
var chart;
const colorsunder = [
  'rgba(255, 99, 132, 0.05)',
  'rgba(99, 255, 132, 0.05)',
  'rgba(99, 132, 225, 0.05)',
  'rgba(0, 99, 132, 0.05)',
  'rgba(255, 0, 132, 0.05)',
]
const colorson = [
  'rgba(255, 99, 132, 0.8)',
  'rgba(99, 255, 132, 0.8)',
  'rgba(99, 132, 225, 0.8)',
  'rgba(0, 99, 132, 0.8)',
  'rgba(255, 0, 132, 0.8)',
]
function enableCharts(elements) {
  var datasets = [];
  dps = [];
  dbl = [];
  for (var i = 0; i < elements; i++) {
    dps.push([]);
    datasets.push({
      label: 'Distance Edge ' + reverseedgeindexmap[i] + ' (m)',
      data: dps[i],
      backgroundColor: colorsunder[i],
      borderColor: colorson[i]
    });
  }
  var chartt = new Chart(document.getElementById('chart1'),
    {
      type: 'line',
      data: {
        datasets: datasets,
        labels: dbl
      },
      options: {
        responsive: false,
        title: { text: 'Edge Node to beacon distance', display: true },
        scales: {
          yAxes: [{
            ticks: {
              suggestedMin: -0.5,
              suggestedMax: 8
            }
          }]
        }
      }
  });
  
  chart = chartt
}

function chartsUpdateDistances(data, inputmap, maxlen) {
  for (var i in data) {
    var cur = data[i];
    var dataset = dps[inputmap[cur.Edge]];
    dataset.push(cur.distance);
    if (cur.Edge == 1) {
      dbl.push(cur.Datetime);
    }

    if (dataset.length > maxlen) {
      dataset.shift();
    }
    if (dbl.length > maxlen) {
      dbl.shift();
    }
  }
  chart.update()
}

// Standard Normal variate using Box-Muller transform.
function randnorm() {
  var u = 0, v = 0;
  while(u === 0) u = Math.random(); //Converting [0,1) to (0,1)
  while(v === 0) v = Math.random();
  return Math.sqrt( -2.0 * Math.log( u ) ) * Math.cos( 2.0 * Math.PI * v );
}

// Mouse target
var target = null;
function Circle(cx, cy, r, moveable, i) {
  moveable == moveable || false;

  var circle = document.createElementNS("http://www.w3.org/2000/svg", 'circle');
  circle.setAttribute('cx', cx);
  circle.setAttribute('cy', cy);
  circle.setAttribute('r', r);
  circle.style.fill = colorson[i];
  circle.owner = this;
  this.element = circle;
  this.x = cx;
  this.y = cy;
  if (moveable) {
    this.element.addEventListener("mousedown", function(event) {
      target = event.target;
    });
  }
}

Circle.prototype.move = function(x, y) {
  this.x = x;
  this.y = y;
  this.element.setAttribute('cx', x);
  this.element.setAttribute('cy', y);
},
Circle.prototype.addToSVG = function(id) {
  var svgwin = document.getElementById(id);
  svgwin.appendChild(this.element);
  return true
}

function setupSVGClick(svgele) {
  var svg = document.getElementById(svgele);
  svg.addEventListener('mousemove', function(event) {
    if (target == null) {
      return;
    }
    target.owner.move(event.offsetX, event.offsetY);
  });
  svg.addEventListener('mouseup', function(event) {
    if (target == null) {
      return;
    }
    target.owner.move(event.offsetX, event.offsetY);
    target = null
  });
}

function getCenterAndMove(nodes, distances, nodemove) {
  var positions = [];
  for (var i in nodes) {
    positions.push({x: nodes[i].x, y: nodes[i].y, distance: distances[i]});
  }
  var loc = getTrilateration(positions);
  if (loc.x == Number.POSITIVE_INFINITY || loc.y == Number.POSITIVE_INFINITY) {
    return;
  }
  nodemove.move(loc.x, loc.y);
};

var circleloc;
var edges = [];
var units = 'cm';

function calculateDistance(block) {
  // Edge, Datetime, Rssi
  var txpower = -70;
  var signalpropconst = 2;
  for (var n in block) {
    block[n].distance = Math.pow(10, 
        (txpower - block[n].Rssi) / (10 * signalpropconst))
  }
  return block
}


// Setup for filters
var filters = [];
function setupFilters(count) {
  filters = [];
  var iirCalculator = new Fili.CalcCascades();
  var coefficients = iirCalculator.lowpass({
        order: 4,
        characteristic: 'butterworth',
        Fs: 10,
        Fc: 0.5,
        gain: 0,
        preGain: false
  });
  for (var i = 0; i < count; i++) {
    filters.push(new Fili.IirFilter(coefficients));
  }
}

function filterDistances(distances, key) {
  for (var i in distances) {
    var filteri = key[distances[i].Edge];
    distances[i].distance = filters[filteri].singleStep(distances[i].distance);
  }
  return distances
}

function applyDistances(actdist) {
  var distances = [];
  for (var key in actdist) {
    distances[edgeindexmap[key]] = actdist[key];
  }
  getCenterAndMove(edges, distances, circleloc);
}

function averageDistances(filtered) {
  var result = {};
  var count = {};
  for (var i in filtered) {
    if (!result[filtered[i].Edge]) {
      result[filtered[i].Edge] = 0;
      count[filtered[i].Edge] = 0;
    }
    result[filtered[i].Edge] += filtered[i].distance;
    count[filtered[i].Edge] += 1;
  }
  var keys = Object.keys(result);
  for (var key in keys) {
    result[keys[key]] /= count[keys[key]];
  }
  return result
}

var helddata = null;
var cursor = 0;
var blocks = []
function processData(data) {
  // On fetch
  if (data) {
    helddata = data;
    cursor = 0;
    blocks = [];
    var second = helddata.map((o) => {
      var date = new Date(o.Datetime);
      return date.getSeconds();
    })
    var cursec = second[0];
    var block = 0;
    blocks.push([]);
    for (var i in second) {
      if (second[i] != cursec) {
        block++;
        blocks.push([]);
        cursec = second[i];
      }
      blocks[block].push(helddata[i])
    }
  }

  if (cursor >= blocks.length) {
    setTimeout(processData, 0);
    return;
  }
  if (justupdated) {
    justupdated = false;
    setTimeout(startLoop, 0);
    return;
  }  
  // Process current block
  var distances = calculateDistance(blocks[cursor]);
  var filtered = distances;
  if (dofilter) {
    filtered = filterDistances(distances, edgeindexmap);
  }

  // Chart update
  chartsUpdateDistances(filtered, edgeindexmap, 50);
  var average = averageDistances(filtered);
  applyDistances(average);
  cursor++;
  if (cursor >= blocks.length) {
    // This should work
    setTimeout(startLoop, 0);
    return;
  }

  setTimeout(processData, 1000);

}

function filterswitch(event) {
  dofilter = event.target.checked;
}

function submitForm(event) {
  var edgeselement = document.getElementById('edges');
  var beaconelement = document.getElementById('beacon');
  var tb = eval(beaconelement.value);
  var te = eval(edgeselement.value);
  addEdges(te);
  beaconid = tb;
}

function reverseMap(map) {
  var keys = Object.keys(map);
  var output = {}
  for (var i in keys) {
    var keyval = keys[i];
    output[map[keyval]] = keyval;
  }
  return output;
}

var edgeindexmap = {};
var reverseedgeindexmap = {};
var edges = [];
var edgenums = [];
var beaconid = 3;
var dofilter = true;
// First doesn't count
var justupdated = true;

function startLoop() {
  var d = new Date();
  // Subtract 5 seconds
  d = new Date(d - 5000);
  var n = d.toISOString();
  n = dateFormat(n, 'isoDateTime');
  // If you change the edges make sure you change the edge index map
  var bodyobj = {
    "Edges": edgenums, 
    "Beacon": beaconid, 
    "Since": n
  };
  fetch(target + '/history/short', {
    method: "POST",
    body: JSON.stringify(bodyobj)
  }).then(function(res) {
    return res.json();
  }).then(function(data) {
    processData(data);
  });
}

function addEdges(lEdgenums) {
  justupdated = !justupdated;
  edgeindexmap = {};
  var svgroot = document.getElementById('svgwin');
  edges.forEach((e) => {
    svgroot.removeChild(e.element);
  });
  edges = [];
  edgenums = lEdgenums;
  for (var i in lEdgenums) {
    edges.push(new Circle(Math.random()*360+20, Math.random()*360+20,
          30, true, i));
    edges[i].addToSVG('svgwin');
    edgeindexmap[lEdgenums[i]] = i;
  }
  reverseedgeindexmap = reverseMap(edgeindexmap);
  enableCharts(edges.length);
  setupFilters(edges.length);
}

function startup() {
  circleloc = new Circle(200, 200, 15);
  circleloc.element.setAttribute('class', 'circle-slide');
  addEdges([1, 2, 3, 4, 5]);
  beaconid = 3;
  circleloc.addToSVG('svgwin');
  setupSVGClick('svgwin');
  document.getElementById('filtercheckbox').addEventListener('change', filterswitch);
  document.getElementById('submit').addEventListener('click', submitForm);
  // Chart update
  startLoop();
}

window.onload = startup;

export {getCenterAndMove, Circle, startLoop}
