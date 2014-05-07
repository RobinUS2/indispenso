// Javascript API helper
function jsApi(method, data, callback, errCallback) {
	$.ajax('/api?method=' + encodeURIComponent(method), {
	    'data': JSON.stringify(data), //{action:'x',params:['a','b','c']}
	    'type': 'POST',
	    'contentType': 'application/json',
	    'error': function(xhr, status, err) {
	    	if (errCallback != null && typeof errCallback === 'function') {
	    		errCallback(xhr, status, err);
	    	}
	    }
	}).done(function( data ) {
		callback(data);
	});
}

/** Show panel */
function showPanel(id) {
	$('div.content-pane[data-id="' + id + '"]').addClass('content-visible');
	return true;
}

// Example handshake
// jsApi('mirror', {a:1}, function(x){console.log(x);});

/** WEB APPLICATION CODE */

/** Login */
$(document).ready(function() {
	/** Are we logged in? */
	jsApi('mirror', {a:1}, function(x){
		/** Yes */
	}, function() {
		/** No */
		showPanel('login');
	});
});