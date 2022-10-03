// import express
const { Router } = require('express');
var express = require('express');
var bodyParser = require('body-parser');
const pgp = require('pg-promise')()
const db = pgp('postgresql://postgres:8dicOKL9d3TpOmQ1U0jV@containers-us-west-85.railway.app:7369/railway')

// set the express server to app
var app = express();

// use the body-parser package to parse bodies?
app.use(bodyParser.urlencoded({
    extended: true
}));

// set port
var port = process.env.PORT || 3000;

// create a router
var router = express.Router();

router.route('/v1/express-animals').get(getAnimals) 


// register routes with api
app.use('/api', router);

// start server
app.listen(port);
console.log('listening on port ' + port);

function getAnimals(req, res) {

    let limit = req.query.limit
    let query = `select * from animals limit ${limit}`
   
    db.many(query)
    .then((data) => {
        res.json({ data: data });
    }) 
    .catch((error) => {
        res.json({ error: error })
    })
 }