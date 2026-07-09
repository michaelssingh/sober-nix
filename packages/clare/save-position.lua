-- save-position.lua
-- Saves the current playback position on quit/pause, using the media-title as the key
-- Note: mal_id, ep_no, and jikan_duration are dynamically prepended to this script by Go.

local utils = require 'mp.utils'

local function get_history_file()
    local dir = os.getenv("CLARE_STATE_DIR") or (os.getenv("XDG_STATE_HOME") or (os.getenv("HOME") .. "/.local/state")) .. "/clare"
    return dir .. "/positions.json"
end

local function load_positions()
    local path = get_history_file()
    local f = io.open(path, "r")
    if not f then return {} end
    local content = f:read("*a")
    f:close()
    local ok, data = pcall(utils.parse_json, content)
    if ok and type(data) == "table" then
        return data
    end
    return {}
end

local function save_positions(positions)
    local path = get_history_file()
    -- Ensure directory exists
    local dir = path:match("(.+)/[^/]+$")
    if dir then
        os.execute('mkdir -p "' .. dir .. '"')
    end
    local f = io.open(path, "w")
    if f then
        f:write(utils.format_json(positions))
        f:close()
    end
end

local skip_intervals = {}
if skip_times_json and skip_times_json ~= "" then
    local ok, decoded = pcall(utils.parse_json, skip_times_json)
    if ok and type(decoded) == "table" then
        skip_intervals = decoded
    end
end

local current_skipped = {}

mp.observe_property("time-pos", "number", function(_, val)
    if val then
        last_time = val
        if auto_skip then
            for _, result in ipairs(skip_intervals) do
                local start_t = result.interval.startTime
                local end_t = result.interval.endTime
                local skip_type = result.skipType
                
                -- Only skip Opening, Ending, and Recaps (preserving Prologues/Epilogues)
                -- Add autoskip_delay seconds of safety pad to the start so pre-OP dialogue is not cut off
                local padded_start = start_t + autoskip_delay
                if (skip_type == "op" or skip_type == "mixed-op" or skip_type == "ed" or skip_type == "mixed-ed" or skip_type == "recap") and val >= padded_start and val < end_t then
                    local skip_key = mal_id .. "_" .. ep_no .. "_" .. skip_type
                    if not current_skipped[skip_key] then
                        current_skipped[skip_key] = true
                        mp.commandv("seek", end_t, "absolute")
                        mp.osd_message("Auto-skipped " .. string.upper(skip_type) .. " (" .. math.floor(end_t - val) .. "s)", 3)
                        break
                    end
                end
            end
        end
    end
end)

local function update_position()
    if not mal_id or mal_id == "" or mal_id == "0" then return end
    local time = last_time or mp.get_property_number("time-pos")
    local duration = mp.get_property_number("duration") or jikan_duration or 1440.0
    if time and duration and duration > 0 then
        local percent = time / duration
        local data = load_positions()
        if not data[mal_id] then
            data[mal_id] = {
                resume_state = nil,
                completed_episodes = {}
            }
        end
        local show = data[mal_id]
        if not show.completed_episodes then
            show.completed_episodes = {}
        end
        
        if percent >= 0.8 then
            show.resume_state = nil
            local found = false
            for _, val in ipairs(show.completed_episodes) do
                if val == ep_no then
                    found = true
                    break
                end
            end
            if not found then
                table.insert(show.completed_episodes, ep_no)
            end
        else
            show.resume_state = {
                episode = ep_no,
                position_seconds = time,
                total_seconds = duration
            }
        end
        
        data[mal_id] = show
        save_positions(data)
    end
end

-- Periodically save every 15 seconds
mp.add_periodic_timer(15, update_position)

mp.register_event("end-file", update_position)
mp.register_event("shutdown", update_position)

-- Hotkeys for skipping intro/outro (85 seconds)
mp.add_key_binding("i", "skip-intro", function()
    mp.commandv("seek", 85, "relative")
    mp.osd_message("Skipped Intro (+85s)", 2)
end)

mp.add_key_binding("ctrl+RIGHT", "skip-intro-ctrl", function()
    mp.commandv("seek", 85, "relative")
    mp.osd_message("Skipped Intro (+85s)", 2)
end)

mp.add_key_binding("o", "skip-outro", function()
    mp.commandv("seek", -85, "relative")
    mp.osd_message("Skipped Outro (-85s)", 2)
end)

mp.add_key_binding("ctrl+LEFT", "skip-outro-ctrl", function()
    mp.commandv("seek", -85, "relative")
    mp.osd_message("Skipped Outro (-85s)", 2)
end)
