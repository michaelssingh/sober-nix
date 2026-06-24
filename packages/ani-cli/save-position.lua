-- save-position.lua
-- Saves the current playback position on quit/pause, using the media-title as the key

local utils = require 'mp.utils'

local function get_history_file()
    local dir = os.getenv("ANI_CLI_HIST_DIR") or (os.getenv("XDG_STATE_HOME") or (os.getenv("HOME") .. "/.local/state")) .. "/ani-cli"
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
        os.execute("mkdir -p " .. utils.shell_escape(dir))
    end
    local f = io.open(path, "w")
    if f then
        f:write(utils.format_json(positions))
        f:close()
    end
end

local function update_position()
    local title = mp.get_property("media-title")
    if title and title ~= "" then
        local time = mp.get_property_number("time-pos")
        local percent = mp.get_property_number("percent-pos")
        if time then
            local positions = load_positions()
            -- If we are near the end (e.g. > 95%), clear the position
            if percent and percent > 95 then
                positions[title] = nil
            else
                positions[title] = time
            end
            save_positions(positions)
        end
    end
end

local function format_time(seconds)
    local m = math.floor(seconds / 60)
    local s = math.floor(seconds % 60)
    return string.format("%d:%02d", m, s)
end

-- On file loaded, check if we have a saved position for this title
mp.register_event("file-loaded", function()
    local title = mp.get_property("media-title")
    if title and title ~= "" then
        local positions = load_positions()
        local pos = positions[title]
        if pos and pos > 0 then
            mp.msg.info("Resuming at position: " .. pos)
            mp.commandv("seek", pos, "absolute")
            mp.osd_message("Resumed at " .. format_time(pos), 3)
        end
    end
end)

-- On pause, also update position
mp.observe_property("pause", "bool", function(name, val)
    update_position()
end)

-- Periodically save every 15 seconds
mp.add_periodic_timer(15, update_position)

mp.register_event("shutdown", update_position)
