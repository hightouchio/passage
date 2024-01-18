// source: service.proto
/**
 * @fileoverview
 * @enhanceable
 * @suppress {missingRequire} reports error on implicit type usages.
 * @suppress {messageConventions} JS Compiler reports an error if a variable or
 *     field starts with 'MSG_' and isn't a translatable message.
 * @public
 */
// GENERATED CODE -- DO NOT EDIT!
/* eslint-disable */
// @ts-nocheck

var jspb = require('google-protobuf');
var goog = jspb;
var global = (function() {
  if (this) { return this; }
  if (typeof window !== 'undefined') { return window; }
  if (typeof global !== 'undefined') { return global; }
  if (typeof self !== 'undefined') { return self; }
  return Function('return this')();
}.call(null));

var google_protobuf_empty_pb = require('google-protobuf/google/protobuf/empty_pb.js');
goog.object.extend(proto, google_protobuf_empty_pb);
var tunnel_pb = require('./tunnel_pb.js');
goog.object.extend(proto, tunnel_pb);
goog.exportSymbol('proto.CreateReverseTunnelRequest', null, global);
goog.exportSymbol('proto.CreateStandardTunnelRequest', null, global);
goog.exportSymbol('proto.DeleteTunnelRequest', null, global);
goog.exportSymbol('proto.GetTunnelRequest', null, global);
goog.exportSymbol('proto.GetTunnelResponse', null, global);
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.CreateStandardTunnelRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.CreateStandardTunnelRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.CreateStandardTunnelRequest.displayName = 'proto.CreateStandardTunnelRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.CreateReverseTunnelRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.CreateReverseTunnelRequest.repeatedFields_, null);
};
goog.inherits(proto.CreateReverseTunnelRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.CreateReverseTunnelRequest.displayName = 'proto.CreateReverseTunnelRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.GetTunnelRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.GetTunnelRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.GetTunnelRequest.displayName = 'proto.GetTunnelRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.GetTunnelResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.GetTunnelResponse.repeatedFields_, null);
};
goog.inherits(proto.GetTunnelResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.GetTunnelResponse.displayName = 'proto.GetTunnelResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.DeleteTunnelRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.DeleteTunnelRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.DeleteTunnelRequest.displayName = 'proto.DeleteTunnelRequest';
}



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.CreateStandardTunnelRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.CreateStandardTunnelRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.CreateStandardTunnelRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.CreateStandardTunnelRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    sshHost: jspb.Message.getFieldWithDefault(msg, 1, ""),
    sshPort: jspb.Message.getFieldWithDefault(msg, 2, 0),
    sshUser: jspb.Message.getFieldWithDefault(msg, 3, ""),
    serviceHost: jspb.Message.getFieldWithDefault(msg, 4, ""),
    servicePort: jspb.Message.getFieldWithDefault(msg, 5, 0),
    createKeyPair: jspb.Message.getBooleanFieldWithDefault(msg, 6, false)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.CreateStandardTunnelRequest}
 */
proto.CreateStandardTunnelRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.CreateStandardTunnelRequest;
  return proto.CreateStandardTunnelRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.CreateStandardTunnelRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.CreateStandardTunnelRequest}
 */
proto.CreateStandardTunnelRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setSshHost(value);
      break;
    case 2:
      var value = /** @type {number} */ (reader.readUint32());
      msg.setSshPort(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setSshUser(value);
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setServiceHost(value);
      break;
    case 5:
      var value = /** @type {number} */ (reader.readUint32());
      msg.setServicePort(value);
      break;
    case 6:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setCreateKeyPair(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.CreateStandardTunnelRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.CreateStandardTunnelRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.CreateStandardTunnelRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.CreateStandardTunnelRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getSshHost();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getSshPort();
  if (f !== 0) {
    writer.writeUint32(
      2,
      f
    );
  }
  f = /** @type {string} */ (jspb.Message.getField(message, 3));
  if (f != null) {
    writer.writeString(
      3,
      f
    );
  }
  f = message.getServiceHost();
  if (f.length > 0) {
    writer.writeString(
      4,
      f
    );
  }
  f = message.getServicePort();
  if (f !== 0) {
    writer.writeUint32(
      5,
      f
    );
  }
  f = message.getCreateKeyPair();
  if (f) {
    writer.writeBool(
      6,
      f
    );
  }
};


/**
 * optional string ssh_host = 1;
 * @return {string}
 */
proto.CreateStandardTunnelRequest.prototype.getSshHost = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.CreateStandardTunnelRequest} returns this
 */
proto.CreateStandardTunnelRequest.prototype.setSshHost = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional uint32 ssh_port = 2;
 * @return {number}
 */
proto.CreateStandardTunnelRequest.prototype.getSshPort = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 2, 0));
};


/**
 * @param {number} value
 * @return {!proto.CreateStandardTunnelRequest} returns this
 */
proto.CreateStandardTunnelRequest.prototype.setSshPort = function(value) {
  return jspb.Message.setProto3IntField(this, 2, value);
};


/**
 * optional string ssh_user = 3;
 * @return {string}
 */
proto.CreateStandardTunnelRequest.prototype.getSshUser = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.CreateStandardTunnelRequest} returns this
 */
proto.CreateStandardTunnelRequest.prototype.setSshUser = function(value) {
  return jspb.Message.setField(this, 3, value);
};


/**
 * Clears the field making it undefined.
 * @return {!proto.CreateStandardTunnelRequest} returns this
 */
proto.CreateStandardTunnelRequest.prototype.clearSshUser = function() {
  return jspb.Message.setField(this, 3, undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.CreateStandardTunnelRequest.prototype.hasSshUser = function() {
  return jspb.Message.getField(this, 3) != null;
};


/**
 * optional string service_host = 4;
 * @return {string}
 */
proto.CreateStandardTunnelRequest.prototype.getServiceHost = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.CreateStandardTunnelRequest} returns this
 */
proto.CreateStandardTunnelRequest.prototype.setServiceHost = function(value) {
  return jspb.Message.setProto3StringField(this, 4, value);
};


/**
 * optional uint32 service_port = 5;
 * @return {number}
 */
proto.CreateStandardTunnelRequest.prototype.getServicePort = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 5, 0));
};


/**
 * @param {number} value
 * @return {!proto.CreateStandardTunnelRequest} returns this
 */
proto.CreateStandardTunnelRequest.prototype.setServicePort = function(value) {
  return jspb.Message.setProto3IntField(this, 5, value);
};


/**
 * optional bool create_key_pair = 6;
 * @return {boolean}
 */
proto.CreateStandardTunnelRequest.prototype.getCreateKeyPair = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 6, false));
};


/**
 * @param {boolean} value
 * @return {!proto.CreateStandardTunnelRequest} returns this
 */
proto.CreateStandardTunnelRequest.prototype.setCreateKeyPair = function(value) {
  return jspb.Message.setProto3BooleanField(this, 6, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.CreateReverseTunnelRequest.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.CreateReverseTunnelRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.CreateReverseTunnelRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.CreateReverseTunnelRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.CreateReverseTunnelRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    publicKeysList: (f = jspb.Message.getRepeatedField(msg, 1)) == null ? undefined : f,
    createKeyPair: jspb.Message.getBooleanFieldWithDefault(msg, 2, false)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.CreateReverseTunnelRequest}
 */
proto.CreateReverseTunnelRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.CreateReverseTunnelRequest;
  return proto.CreateReverseTunnelRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.CreateReverseTunnelRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.CreateReverseTunnelRequest}
 */
proto.CreateReverseTunnelRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.addPublicKeys(value);
      break;
    case 2:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setCreateKeyPair(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.CreateReverseTunnelRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.CreateReverseTunnelRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.CreateReverseTunnelRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.CreateReverseTunnelRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getPublicKeysList();
  if (f.length > 0) {
    writer.writeRepeatedString(
      1,
      f
    );
  }
  f = message.getCreateKeyPair();
  if (f) {
    writer.writeBool(
      2,
      f
    );
  }
};


/**
 * repeated string public_keys = 1;
 * @return {!Array<string>}
 */
proto.CreateReverseTunnelRequest.prototype.getPublicKeysList = function() {
  return /** @type {!Array<string>} */ (jspb.Message.getRepeatedField(this, 1));
};


/**
 * @param {!Array<string>} value
 * @return {!proto.CreateReverseTunnelRequest} returns this
 */
proto.CreateReverseTunnelRequest.prototype.setPublicKeysList = function(value) {
  return jspb.Message.setField(this, 1, value || []);
};


/**
 * @param {string} value
 * @param {number=} opt_index
 * @return {!proto.CreateReverseTunnelRequest} returns this
 */
proto.CreateReverseTunnelRequest.prototype.addPublicKeys = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 1, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.CreateReverseTunnelRequest} returns this
 */
proto.CreateReverseTunnelRequest.prototype.clearPublicKeysList = function() {
  return this.setPublicKeysList([]);
};


/**
 * optional bool create_key_pair = 2;
 * @return {boolean}
 */
proto.CreateReverseTunnelRequest.prototype.getCreateKeyPair = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 2, false));
};


/**
 * @param {boolean} value
 * @return {!proto.CreateReverseTunnelRequest} returns this
 */
proto.CreateReverseTunnelRequest.prototype.setCreateKeyPair = function(value) {
  return jspb.Message.setProto3BooleanField(this, 2, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.GetTunnelRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.GetTunnelRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.GetTunnelRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.GetTunnelRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    id: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.GetTunnelRequest}
 */
proto.GetTunnelRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.GetTunnelRequest;
  return proto.GetTunnelRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.GetTunnelRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.GetTunnelRequest}
 */
proto.GetTunnelRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setId(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.GetTunnelRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.GetTunnelRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.GetTunnelRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.GetTunnelRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string id = 1;
 * @return {string}
 */
proto.GetTunnelRequest.prototype.getId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.GetTunnelRequest} returns this
 */
proto.GetTunnelRequest.prototype.setId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.GetTunnelResponse.repeatedFields_ = [2];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.GetTunnelResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.GetTunnelResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.GetTunnelResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.GetTunnelResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    tunnel: (f = msg.getTunnel()) && tunnel_pb.Tunnel.toObject(includeInstance, f),
    instancesList: jspb.Message.toObjectList(msg.getInstancesList(),
    tunnel_pb.TunnelInstance.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.GetTunnelResponse}
 */
proto.GetTunnelResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.GetTunnelResponse;
  return proto.GetTunnelResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.GetTunnelResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.GetTunnelResponse}
 */
proto.GetTunnelResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new tunnel_pb.Tunnel;
      reader.readMessage(value,tunnel_pb.Tunnel.deserializeBinaryFromReader);
      msg.setTunnel(value);
      break;
    case 2:
      var value = new tunnel_pb.TunnelInstance;
      reader.readMessage(value,tunnel_pb.TunnelInstance.deserializeBinaryFromReader);
      msg.addInstances(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.GetTunnelResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.GetTunnelResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.GetTunnelResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.GetTunnelResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getTunnel();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      tunnel_pb.Tunnel.serializeBinaryToWriter
    );
  }
  f = message.getInstancesList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      2,
      f,
      tunnel_pb.TunnelInstance.serializeBinaryToWriter
    );
  }
};


/**
 * optional Tunnel tunnel = 1;
 * @return {?proto.Tunnel}
 */
proto.GetTunnelResponse.prototype.getTunnel = function() {
  return /** @type{?proto.Tunnel} */ (
    jspb.Message.getWrapperField(this, tunnel_pb.Tunnel, 1));
};


/**
 * @param {?proto.Tunnel|undefined} value
 * @return {!proto.GetTunnelResponse} returns this
*/
proto.GetTunnelResponse.prototype.setTunnel = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.GetTunnelResponse} returns this
 */
proto.GetTunnelResponse.prototype.clearTunnel = function() {
  return this.setTunnel(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.GetTunnelResponse.prototype.hasTunnel = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * repeated TunnelInstance instances = 2;
 * @return {!Array<!proto.TunnelInstance>}
 */
proto.GetTunnelResponse.prototype.getInstancesList = function() {
  return /** @type{!Array<!proto.TunnelInstance>} */ (
    jspb.Message.getRepeatedWrapperField(this, tunnel_pb.TunnelInstance, 2));
};


/**
 * @param {!Array<!proto.TunnelInstance>} value
 * @return {!proto.GetTunnelResponse} returns this
*/
proto.GetTunnelResponse.prototype.setInstancesList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 2, value);
};


/**
 * @param {!proto.TunnelInstance=} opt_value
 * @param {number=} opt_index
 * @return {!proto.TunnelInstance}
 */
proto.GetTunnelResponse.prototype.addInstances = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 2, opt_value, proto.TunnelInstance, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.GetTunnelResponse} returns this
 */
proto.GetTunnelResponse.prototype.clearInstancesList = function() {
  return this.setInstancesList([]);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.DeleteTunnelRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.DeleteTunnelRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.DeleteTunnelRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.DeleteTunnelRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    id: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.DeleteTunnelRequest}
 */
proto.DeleteTunnelRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.DeleteTunnelRequest;
  return proto.DeleteTunnelRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.DeleteTunnelRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.DeleteTunnelRequest}
 */
proto.DeleteTunnelRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setId(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.DeleteTunnelRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.DeleteTunnelRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.DeleteTunnelRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.DeleteTunnelRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string id = 1;
 * @return {string}
 */
proto.DeleteTunnelRequest.prototype.getId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.DeleteTunnelRequest} returns this
 */
proto.DeleteTunnelRequest.prototype.setId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


goog.object.extend(exports, proto);
