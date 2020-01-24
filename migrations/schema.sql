--
-- PostgreSQL database dump
--

-- Dumped from database version 12.1
-- Dumped by pg_dump version 12.1

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: brigades; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.brigades (
    id integer NOT NULL,
    guild_id text NOT NULL,
    active_project_category_id text NOT NULL,
    inactive_project_category_id text NOT NULL,
    new_user_role text NOT NULL,
    onboarding_role text NOT NULL,
    member_role text NOT NULL,
    onboarding_invite_code text NOT NULL,
    onboarding_invite_count integer NOT NULL,
    code_of_conduct_message_id text NOT NULL,
    agenda_folder_id text NOT NULL,
    timezone_string text NOT NULL,
    github_organization text NOT NULL,
    issue_emoji text NOT NULL
);


ALTER TABLE public.brigades OWNER TO postgres;

--
-- Name: files; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.files (
    id integer NOT NULL,
    brigade_id integer NOT NULL,
    name text NOT NULL,
    url text NOT NULL
);


ALTER TABLE public.files OWNER TO postgres;

--
-- Name: meetings; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.meetings (
    id integer NOT NULL,
    brigade_id integer NOT NULL,
    date timestamp without time zone NOT NULL,
    attendance_count integer NOT NULL
);


ALTER TABLE public.meetings OWNER TO postgres;

--
-- Name: schema_migration; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.schema_migration (
    version character varying(14) NOT NULL
);


ALTER TABLE public.schema_migration OWNER TO postgres;

--
-- Name: volunteer_sessions; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.volunteer_sessions (
    id integer NOT NULL,
    brigade_id integer NOT NULL,
    discord_user_id text NOT NULL,
    start_time timestamp without time zone NOT NULL,
    duration integer
);


ALTER TABLE public.volunteer_sessions OWNER TO postgres;

--
-- Name: brigades brigades_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.brigades
    ADD CONSTRAINT brigades_pkey PRIMARY KEY (id);


--
-- Name: files files_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.files
    ADD CONSTRAINT files_pkey PRIMARY KEY (id);


--
-- Name: meetings meetings_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.meetings
    ADD CONSTRAINT meetings_pkey PRIMARY KEY (id);


--
-- Name: volunteer_sessions volunteer_sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.volunteer_sessions
    ADD CONSTRAINT volunteer_sessions_pkey PRIMARY KEY (id);


--
-- Name: schema_migration_version_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX schema_migration_version_idx ON public.schema_migration USING btree (version);


--
-- Name: files files_brigade_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.files
    ADD CONSTRAINT files_brigade_id_fkey FOREIGN KEY (brigade_id) REFERENCES public.brigades(id);


--
-- Name: meetings meetings_brigade_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.meetings
    ADD CONSTRAINT meetings_brigade_id_fkey FOREIGN KEY (brigade_id) REFERENCES public.brigades(id);


--
-- Name: volunteer_sessions volunteer_sessions_brigade_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.volunteer_sessions
    ADD CONSTRAINT volunteer_sessions_brigade_id_fkey FOREIGN KEY (brigade_id) REFERENCES public.brigades(id);


--
-- PostgreSQL database dump complete
--

