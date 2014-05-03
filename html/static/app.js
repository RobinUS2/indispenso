// Javascript API helper
function jsApi(method, data, callback) {
	$.ajax('/api?method=' + encodeURIComponent(method), {
	    'data': JSON.stringify(data), //{action:'x',params:['a','b','c']}
	    'type': 'POST',
	    'contentType': 'application/json'
	}).done(function( data ) {
		callback(data);
	});
}

// Test handshake
jsApi('mirror', {a:1}, function(x){console.log(x);});