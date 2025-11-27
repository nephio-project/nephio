from flask import Flask, request, jsonify
from kubernetes import client, config
from kubernetes.client.rest import ApiException
import os
import logging
from datetime import datetime
from controllers.utils import validate_cluster_creation_request
app = Flask(__name__)

@app.route('/O2ims_infrastructureProvisioning/v1/provisioningRequests ', methods=['POST'])
def trigger_action():
    data = request.json
    logging.info("O2IMS API Received Request Payload Is:", data)
    # add validation logic here
    now = datetime.now()
    dt_string = now.strftime("%Y-%m-%d %H:%M:%S")   
    try:
        validate_cluster_creation_request(params=data)
        o2ims_cr={
                'apiVersion': 'o2ims.provisioning.oran.org/v1alpha1',
                'kind': 'ProvisioningRequest',
                'metadata': {
                    'name': data.get('name'),
                    'labels':{
                        'provisioningRequestId': data.get('provisioningRequestId')
                    }
                },
                'spec':{
                    'description': data.get('description'),
                    'name':  data.get('name'),
                    'templateName': data.get('templateName'),
                    'templateParameters':data.get('templateParameters'),
                    'templateVersion': data.get('templateVersion')
                }   
        }
        logging.debug("O2IMS CR Payload Is:", o2ims_cr)
        config.load_incluster_config()
        api = client.CustomObjectsApi()
        response = api.create_cluster_custom_object(
                    group='o2ims.provisioning.oran.org',
                    version='v1alpha1',
                    plural='provisioningrequest',
                    body=o2ims_cr
        )
    except Exception as e:
        logging.error(f"Caught Exception while deploying O2IMS CR ,{e}")
        return jsonify({"status":{"updateTime":dt_string,"message":f"O2IMS Deployment Failed,{e}","provisioningPhase":"FAILED"}}),500
    return jsonify({"provisioningRequestData": data, "status": {"updateTime":dt_string,"message":"In-Progress","provisioningPhase":"PROGRESSING"},"ProvisionedResourceSet":{"nodeClusterId":"test","infrastructureResourceIds":"sample"}}), 200

@app.route('/O2ims_infrastructureProvisioning/v1/provisioningRequests ', methods=['GET'])
def fetch_status():
    now = datetime.now()
    dt_string = now.strftime("%Y-%m-%d %H:%M:%S")
    logging.info("Received O2IMS GET STATUS API CALL  At %s:",dt_string)

    try:
        config.load_incluster_config()
        api = client.CustomObjectsApi()
        response = api.list_cluster_custom_object(
                    group='o2ims.provisioning.oran.org',
                    version='v1alpha1',
                    plural='provisioningrequest'
        )
        data=response.get('items')
        if len(data)==0:
            status=jsonify({"status":{"updateTime":dt_string,"message":"No ProvisioningRequest Found","provisioningPhase":"FAILED"}})
            response_data={"provisioningRequestData": {}, "status": status,"ProvisionedResourceSet":{}}
            return response_data, 200
        #read all the provisioning requests and create response array from it
        
        for o2ims_cr in data:
            status={}
        #read o2ims_cr status and update message accordingly
            if 'status' in o2ims_cr.keys():
                if o2ims_cr['status'].get('provisioningState')=='failed':
                    status= jsonify({"status":{"updateTime":dt_string,"message":o2ims_cr['status'].get('provisioningMessage'),"provisioningPhase":"FAILED"}})  
                elif o2ims_cr['status'].get('provisioningState')=='progressing':
                    status= jsonify({"status":{"updateTime":dt_string,"message":o2ims_cr['status'].get('provisioningMessage'),"provisioningPhase":"PROGRESSING"}})
                elif o2ims_cr['status'].get('provisioningState')=='fulfilled':
                    status= jsonify({"status":{"updateTime":dt_string,"message":o2ims_cr['status'].get('provisioningMessage'),"provisioningPhase":"FULFILLED"}})
                else:
                    status=({"status":{"updateTime":dt_string,"message":"In-Progress","provisioningPhase":"PROGRESSING"}})
            # read o2ims_cr spec and update response accordingly
            spec=o2ims_cr.get('spec')
            provisionedresourceset={"nodeClusterId":"test","infrastructureResourceIds":"sample"}
            response_data={"provisioningRequestData": spec, "status": status,"ProvisionedResourceSet":provisionedresourceset}
            return response_data, 200
    except client.exceptions.ApiException as e:
        logging.error(f"Caught Exception while fetching O2IMS CR Status ,{e}")
        return jsonify({"status":{"updateTime":dt_string,"message":f"O2IMS Deployment Failed,{e}","provisioningPhase":"FAILED"}}),500
   