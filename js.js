var Loaded = 0;
var Results;
var ResultsPage = document.getElementById("results");
var textInput = document.getElementById("query");
var form = document.getElementsByTagName("form");
var combo = document.getElementById("order");
var topButton = document.getElementById("topButton");
var Months = {};
Months["01"] = "January"
Months["02"] = "February"
Months["03"] = "March"
Months["04"] = "April"
Months["05"] = "May"
Months["06"] = "June"
Months["07"] = "July"
Months["08"] = "August"
Months["09"] = "September"
Months["10"] = "October"
Months["11"] = "November"
Months["12"] = "December"

function start(r){
	if (r.responseText == '') return;
	try {
		Results = JSON.parse(r.responseText);
	} catch(_) {
		return;
	};
	refresh();
	if (Results.strips.length == 0) {
		warn();
	} else {
		var jumbo = document.getElementById("jumbotron");
		if (jumbo) {
			jumbo.remove();
		}
		sort();
		load();
	}
}

function query(searchTerm) {
	if (searchTerm === "") {
		if(Loaded > 0){
			refresh();
			warn();
		}
		return;
	}
	var r = new XMLHttpRequest();
	r.open('post', '/q', true);
	r.send(searchTerm);
	r.onreadystatechange = function(){start(r)};
}

function warn() {
	var pElement = document.createElement("p");
	pElement.textContent = "No strips found.";
	ResultsPage.appendChild(pElement);
	Loaded = 0;
}

function desc(a, b){ return a.date < b.date ? 1 : (a.date > b.date ? -1 : 0); }

function asc(a, b){ return a.date < b.date ? -1 : (a.date > b.date ? 1 : 0); }

function rel(a, b){ return a.rel < b.rel ? -1 : (a.rel > b.rel ? 1 : 0); }

function sort() {
	switch(combo.value) {
		case '1':
			Results.strips = Results.strips.sort(desc);
			break;
		case '-1':
			Results.strips = Results.strips.sort(asc);
			break;
		case '0':
			Results.strips = Results.strips.sort(rel);
			break;
	}
}

function img(name){
	var img = document.createElement("img")
	img.id = name;
	img.src = "/i/"+name.replace(/-/g, "");
	img.onclick = open;
	return img;
}

function load() {
	if (Results == null) {
		return;
	}
	var max = Results.strips.length;
	do {
		for (var i = Loaded; i < Loaded+32 && i < max ; i++) {
			var pic = img(Results.strips[i].date);
			ResultsPage.appendChild(pic);
		}
	} while (false);
	Loaded = i;
}

function refresh() {
	while (ResultsPage.hasChildNodes()){
		ResultsPage.removeChild(ResultsPage.lastChild);
	}
	Loaded = 0;
}

function shrink(e){
	var id = e.target.id;
	id = id.replace('x-', '');
	ResultsPage.insertBefore(img(id), e.target.parentNode);
	e.target.parentNode.remove();
}

function translate(d) {
	var portions = d.split("-");
	var month = portions[1];
	return Months[month]+" "+portions[2]+", "+portions[0];
}

function open(e){
	var section = document.createElement('section');
	var img = document.createElement('img');
	//var aside = highlight(r.responseText);
	var a = document.createElement('a');
	a.textContent = translate(e.target.id);
	a.href = 'https://www.gocomics.com/peanuts/'+
		e.target.id.replace(/-/g, '/');
	img.id = 'x-'+e.target.id;
	img.onclick = shrink;
	img.src = "/I/"+e.target.id.replace(/-/g, "");
	section.appendChild(a);
	//section.appendChild(aside);
	section.appendChild(img);
	var i = document.getElementById(e.target.id);
	if (i != null) i.parentNode.replaceChild(section, i);
	section.scrollIntoView();
}

function titilize(t){
	if (t === "") {
		window.history.pushState('obj', 'newtitle', '/');
	} else {
		window.history.pushState('obj', 'newtitle', '?q='+t);
	}

}

document.addEventListener('scroll', function(e)
{
	if((window.innerHeight + window.pageYOffset + 10)
	>= document.body.offsetHeight){load();}
    if (document.body.scrollTop > 20 || document.documentElement.scrollTop > 20) {
      topButton.style.display = "block";
    } else {
      topButton.style.display = "none";
    }
});

var timeout = null;

textInput.onkeyup = function(e) {
	clearTimeout(timeout);
	timeout = setTimeout(function(){
		titilize(textInput.value);
		query(textInput.value);
	}, 500);
}

window.onload = function(e) {
	window.href = "test";
	if(!(textInput.value === "")){
		query(textInput.value);
	}
}

combo.onchange = function(e) {
	refresh();
	sort();
	load();
}

form[0].onsubmit = function(e) {
	e.preventDefault();
}

topButton.onclick = function(e) {
  document.body.scrollTop = 0;
  document.documentElement.scrollTop = 0;
}
