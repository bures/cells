/*
 * Copyright 2007-2017 Charles du Jeu - Abstrium SAS <team (at) pyd.io>
 * This file is part of Pydio.
 *
 * Pydio is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * Pydio is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with Pydio.  If not, see <http://www.gnu.org/licenses/>.
 *
 * The latest code can be found at <https://pydio.com>.
 */

import React from 'react'
import {Paper, List, ListItem, RaisedButton, Checkbox, Divider, Subheader} from 'material-ui'
import PydioApi from 'pydio/http/api'
import {UpdateServiceApi} from 'pydio/http/rest-api'
import Pydio from 'pydio'
const {moment, SingleJobProgress} = Pydio.requireLib('boot');
import ServiceExposedConfigs from '../core/ServiceExposedConfigs'

const UpdaterDashboard = React.createClass({

    mixins:[AdminComponents.MessagesConsumerMixin],


    getInitialState: function(){
        const {pydio} = this.props;
        return {
            check: -1,
            backend:pydio.Parameters.get("backend")
        };
    },

    componentDidMount:function(){
        this.checkForUpgrade();
    },

    checkForUpgrade: function(){
        const {pydio} = this.props;
        this.setState({loading:true});

        let url = pydio.Parameters.get('ENDPOINT_REST_API') + '/frontend/bootconf';
        window.fetch(url, {
            method:'GET',
            credentials:'same-origin',
        }).then((response) => {
            response.json().then((data) => {
                if(data.backend){
                    this.setState({backend:data.backend})
                }
            }).catch(()=>{});
        }).catch(()=>{});

        const api = new UpdateServiceApi(PydioApi.getRestClient());
        Pydio.startLoading();
        api.updateRequired().then(res => {
            Pydio.endLoading();
            let hasBinary = 0;
            if (res.AvailableBinaries) {
                hasBinary = res.AvailableBinaries.length;
                this.setState({packages: res.AvailableBinaries});
            } else {
                this.setState({no_upgrade: true});
            }
            const node = pydio.getContextNode();
            node.getMetadata().set('flag', hasBinary);
            AdminComponents.MenuItemListener.getInstance().notify("item_changed");
            this.setState({loading: false});
        }).catch(() => {
            Pydio.endLoading();
            this.setState({loading: false});
        });

    },

    upgradeFinished(){
        const {pydio} = this.props;
        this.setState({updateApplied: this.state.selectedPackage.Version});
        const node = pydio.getContextNode();
        node.getMetadata().set('flag', 0);
        AdminComponents.MenuItemListener.getInstance().notify("item_changed");
    },

    performUpgrade: function(){
        const {pydio} = this.props;
        const {check, packages} = this.state;

        if(check < 0 || !packages[check]){
            alert('Please select at least one package!');
            return;
        }

        if(confirm(this.context.getMessage('15', 'updater'))){

            const toApply = packages[check];
            const version = toApply.Version;
            const api = new UpdateServiceApi(PydioApi.getRestClient());
            api.applyUpdate(version).then(res => {
                if (res.Success) {
                    this.setState({watchJob: res.Message});
                } else {
                    pydio.UI.displayMessage('ERROR', res.Message);
                }
            }).finally(()=>{

            });

        }
    },

    onCheckStateChange: function(index, value, pack){
        if(value) {
            this.setState({check: index, selectedPackage: pack});
        } else {
            this.setState({check: -1, selectedPackage: null});
        }
    },

    render:function(){

        let list = null;
        const {packages, check, loading, dirty, updateApplied, selectedPackage, watchJob, backend} = this.state;
        const subHeaderStyle = {
            backgroundColor: '#f5f5f5',
            color: '#9e9e9e',
            fontSize: 12,
            fontWeight: 500,
            borderBottom: '1px solid #e0e0e0',
            height: 48,
            lineHeight: '48px',
            padding: '0 16px'
        };


        let buttons = [];
        if(packages){
            buttons.push(<RaisedButton disabled={check < 0 || updateApplied} secondary={true} label={this.context.getMessage('4', 'updater')} onTouchTap={this.performUpgrade}/>);
            let items = [];
            for (let index=packages.length - 1; index >= 0; index--) {
                const p = packages[index];
                items.push(<ListItem
                    leftCheckbox={<Checkbox key={p} onCheck={(e,v)=> this.onCheckStateChange(index, v, p)} checked={check >= index} disabled={updateApplied || check > index} />}
                    primaryText={p.PackageName + ' ' + p.Version}
                    secondaryText={p.Label + ' - ' + moment(new Date(p.ReleaseDate * 1000)).fromNow()}
                />);
                items.push(<Divider/>)
            }
            items.pop();
            list = (
                <div>
                    <div style={subHeaderStyle}>{this.context.getMessage('16', 'updater')}</div>
                    <List>
                        {items}
                    </List>
                </div>
            );
        }else if(loading){
            list = (
                <div>
                    <div style={subHeaderStyle}>{this.context.getMessage('16', 'updater')}</div>
                    <div style={{padding: 16}}>{this.context.getMessage('17', 'updater')}</div>
                </div>
            );
        }else{
            list = (
                <div style={{minHeight: 36}}>
                    <div style={subHeaderStyle}>{this.context.getMessage('20', 'updater')}</div>
                    <div style={{padding: '16px 16px 32px'}}>
                        <span style={{float:'right'}}>
                            <RaisedButton secondary={true} label={this.context.getMessage('20', 'updater')} onTouchTap={this.checkForUpgrade}/>
                        </span>
                        { (this.state && this.state.no_upgrade) ? this.context.getMessage('18', 'updater') : this.context.getMessage('19', 'updater') }
                    </div>
                </div>
            );
        }

        if (dirty){
            buttons.push(<RaisedButton style={{marginLeft: 10}} secondary={true} label={"Save Configs"} onTouchTap={()=>{
                this.refs.serviceConfigs.save().then((res) => {
                    this.setState({dirty: false});
                });
            }}/>)
        }

        return (
            <div className={"main-layout-nav-to-stack vertical-layout people-dashboard"}>
                <AdminComponents.Header
                    title={this.context.getMessage('2', 'updater')}
                    icon="mdi mdi-update"
                    actions={buttons}
                    reloadAction={()=>{this.checkForUpgrade()}}
                    loading={loading}
                />
                <div style={{flex: 1, overflow: 'auto'}}>
                    <Paper style={{margin:16}} zDepth={1}>
                        <div style={subHeaderStyle}>Current Version</div>
                        <List style={{padding: '0 16px'}}>
                            <ListItem primaryText={backend.PackageLabel + ' ' + backend.Version} disabled={true} secondaryTextLines={2} secondaryText={<span>
                                {"Released : " + backend.BuildStamp}<br/>
                                {"Revision : " + backend.BuildRevision}
                            </span>}/>
                        </List>
                    </Paper>
                    {watchJob &&
                        <Paper style={{margin:'0 16px', position:'relative'}} zDepth={1}>
                            <div style={subHeaderStyle}>{selectedPackage ? (selectedPackage.PackageName + ' ' + selectedPackage.Version) : ''}</div>
                            <div style={{padding:16}}>
                                <SingleJobProgress
                                    jobID={watchJob}
                                    progressStyle={{paddingTop: 16}}
                                    lineStyle={{userSelect:'text'}}
                                    onEnd={()=>{this.upgradeFinished()}}
                                />
                            </div>
                        </Paper>
                    }
                    {!watchJob && list &&
                        <Paper style={{margin:'0 16px', position:'relative'}} zDepth={1}>{list}</Paper>
                    }
                    {!watchJob &&
                        <ServiceExposedConfigs
                            className={"row-flex"}
                            serviceName={"pydio.grpc.update"}
                            ref={"serviceConfigs"}
                            onDirtyChange={(d)=>this.setState({dirty: d})}
                        />
                    }
                </div>
            </div>

        );
    }

});

export {UpdaterDashboard as default}