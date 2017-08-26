import PropTypes from "prop-types";
import React from "react";
import { Button, Card, Divider, Grid, Icon, Label } from "semantic-ui-react";
import _ from "lodash";
import { AddSite } from "./AddSite";
import { DeleteSite } from "./DeleteSite";
import "./site.css";

const SiteCard = props =>
  <Card fluid style={{ textAlign: "center" }}>
    <Card.Header>
      <h2 style={{ margin: "5px" }}>
        {props.site.name}
      </h2>
    </Card.Header>
    <Card.Content style={{ textAlign: "left", fontSize: "16px" }}>
      <p>
        <strong>
          <Icon name="git" />
          {"Repo: "}
        </strong>
      </p>
      <p style={{ overflow: "hidden", textOverflow: "ellipsis" }}>
        {props.site.git_repo}
      </p>
      <p>
        <strong>
          <Icon name="fork" />Branch:{" "}
        </strong>
        {props.site.branch}
      </p>
      <p>
        <strong>
          <Icon name="folder" />Root:{" "}
        </strong>
        {props.site.root}
      </p>
      <p>
        <strong>
          <Icon name="heartbeat" />Status:{" "}
        </strong>
        {props.site.healthy
          ? <Label basic size="tiny" color="green" content="healthy" />
          : <Label basic size="tiny" color="red" content="unhealthy" />}
      </p>
      <Divider />
      <Button.Group widths={8} vertical={true}>
        {_.map(props.site.hostnames, hostname =>
          <Button
            key={hostname}
            onClick={() => {
              window.location = `${window.location.protocol}//${hostname}`;
            }}
            content={hostname}
            icon="globe"
          />
        )}
        <DeleteSite afterDelete={props.onDelete} site={props.site} />
      </Button.Group>
    </Card.Content>
  </Card>;

SiteCard.propTypes = {
  site: PropTypes.object.isRequired,
  siteID: PropTypes.string.isRequired,
  onDelete: PropTypes.func.isRequired
};

export class Sites extends React.Component {
  state = {
    sites: null,
    wildCard: ""
  };

  reloadData = async () => {
    const req = await fetch("/v1/sites");
    const data = await req.json();
    this.setState({ sites: data.static_dirs, wildCard: data.wild_card });
  };

  componentDidMount = async () => {
    return this.reloadData();
  };

  render = () =>
    <Grid>
      <Grid.Column computer={4} mobile={16}>
        <AddSite wildCard={this.state.wildCard} reload={this.reloadData} />
      </Grid.Column>
      <Grid.Column computer={12} tablet={8} mobile={16}>
        <h2>Sites</h2>
        <Grid>
          {_.map(this.state.sites, (site, id) =>
            <Grid.Column computer={8} tablet={16} mobile={16} key={id}>
              <SiteCard onDelete={this.reloadData} site={site} siteID={id} />
            </Grid.Column>
          )}
        </Grid>
      </Grid.Column>
    </Grid>;
}
