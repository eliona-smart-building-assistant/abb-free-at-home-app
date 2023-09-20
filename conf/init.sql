--  This file is part of the eliona project.
--  Copyright © 2022 LEICOM iTEC AG. All Rights Reserved.
--  ______ _ _
-- |  ____| (_)
-- | |__  | |_  ___  _ __   __ _
-- |  __| | | |/ _ \| '_ \ / _` |
-- | |____| | | (_) | | | | (_| |
-- |______|_|_|\___/|_| |_|\__,_|
--
--  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING
--  BUT NOT LIMITED  TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
--  NON INFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM,
--  DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
--  OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

create schema if not exists abb_free_at_home;

-- Should be editable by eliona frontend.
create table if not exists abb_free_at_home.configuration
(
	id               bigserial primary key,
	is_cloud         boolean not null default false,

	client_id        text,
	client_secret    text,

	access_token     text,
	refresh_token    text,
	expiry           timestamp,

	api_url          text,
	api_username     text,
	api_password     text,

	refresh_interval integer not null default 60,
	request_timeout  integer not null default 120,
	asset_filter     json,
	active           boolean default false,
	enable           boolean default false,
	project_ids      text[]
);

create table if not exists abb_free_at_home.asset
(
	id               bigserial primary key,
	configuration_id bigserial not null references abb_free_at_home.configuration(id),
	project_id       text      not null,
	global_asset_id  text      not null,
	asset_type_name  text      not null,
	asset_id         integer   unique
);

create table if not exists abb_free_at_home.datapoint
(
	id                 bigserial primary key,
	asset_id           integer not null references abb_free_at_home.asset(asset_id),
	system_id          text not null,
	device_id          text not null,
	channel_id         text not null,
	datapoint          text not null,
	function           text not null,
	is_input           boolean not null,
	last_written_value integer,
	last_written_time  timestamp with time zone
);

create table if not exists abb_free_at_home.datapoint_attribute
(
	id             bigserial primary key,
	datapoint_id   bigserial not null references abb_free_at_home.datapoint(id),
	subtype        text not null,
	attribute_name text not null
);

-- Makes the new objects available for all other init steps
commit;
