<script>
        import {
                Greet, CreateSchool, ListSchools, DeleteSchool, SchoolHasLessons, SchoolHasSchedule,
                CreateTeacher, ListTeachers, CreateSubject, ListSubjects,
                CreateClass, ListClasses, CreateRoom, ListRooms,
                CreateLesson, ListLessons, DeleteLesson, UpdateLesson,
                CreateConstraint, ListConstraints, DeleteConstraint,
                DeleteTeacher, DeleteSubject, DeleteClass, DeleteRoom, DeleteScheduleEntry, SaveExport,
                UpdateTeacher, UpdateSubject, UpdateClass, UpdateRoom,
                Generate, MoveEntry, MoveEntries, SwapEntries, ReplaceSchedule, ListSchedule, ExportAll, ImportAll, ScheduleCSV, GetSchoolSettings, UpdateSchoolSettings, HasPreciseSolver, ExportPDF, StartPreciseGeneration, GetGenerationProgress
        } from "../wailsjs/go/main/App";
        import { onMount } from "svelte";
        import AppSidebar from "./components/AppSidebar.svelte";
        import ScheduleGrid from "./components/ScheduleGrid.svelte";
        import PlacementSummary from "./components/PlacementSummary.svelte";
        import IssuesTab from "./components/IssuesTab.svelte";
        import ReferencesTab from "./components/ReferencesTab.svelte";
        import LessonsTab from "./components/LessonsTab.svelte";
        import ConstraintsTab from "./components/ConstraintsTab.svelte";
        import SettingsTab from "./components/SettingsTab.svelte";

        let schools = [];
        let activeSchoolID = 0;
        let newSchoolName = "Моя школа";
        let tab = "refs";
        const APP_VERSION = "1.11.6";
        let msg = "";

        let teachers = [], subjects = [], classes = [], rooms = [], lessons = [], constraints = [], schedule = [];

        // O(1)-резолверы имён: раньше subjName/teachName/... делали
        // Array.find по справочнику на КАЖДУЮ ячейку сетки (сотни тысяч
        // сканов на большом расписании). Сигнатуры (list, id) сохранены —
        // список остаётся синтаксической зависимостью для реактивности.
        let subjectById = new Map(), teacherById = new Map(), classById = new Map(), roomById = new Map();
        $: {
                subjectById = new Map(subjects.map((s) => [s.id, s]));
                teacherById = new Map(teachers.map((t) => [t.id, t]));
                classById = new Map(classes.map((c) => [c.id, c]));
                roomById = new Map(rooms.map((r) => [r.id, r]));
        }
        // Подгруппы класса: classID → [id подгрупп]. Для cellSubs —
        // раньше фильтровали весь список классов на каждый вызов.
        let kidsOf = new Map();
        $: {
                const m = new Map();
                for (const c of classes) {
                        if (c.subgroup_of) {
                                if (!m.has(c.subgroup_of)) m.set(c.subgroup_of, []);
                                m.get(c.subgroup_of).push(c.id);
                        }
                }
                kidsOf = m;
        }

        // form models
        let t = { name: "", short_name: "" };
        let s = { name: "", short_name: "", requires_room_type: "any" };
        let c = { name: "", room_id: 0, subgroup_of: null };
        let r = { name: "", room_type: "any" };
        let l = { class_id: 0, subject_id: 0, teacher_id: 0, hours_per_week: 1 };
        let curClass = 0;
        // Инлайн-редактирование справочников: ✎ переводит строку в режим
        // правки, ✓ сохраняет в БД, ✗ откатывает (перезагрузкой справочников).
        let editing = null; // { kind, id }
        function startEdit(kind, id) { editing = { kind, id }; }
        async function cancelEdit() { editing = null; await reloadRefs(); }
        async function saveEdit(kind, x) {
                if (kind === "teacher") await UpdateTeacher({ id: x.id, school_id: x.school_id, name: x.name, short_name: x.short_name });
                else if (kind === "subject") await UpdateSubject({ id: x.id, school_id: x.school_id, name: x.name, short_name: x.short_name, requires_room_type: x.requires_room_type || "any" });
	                else if (kind === "class") await UpdateClass({ id: x.id, school_id: x.school_id, name: x.name, room_id: x.room_id || 0, subgroup_of: x.subgroup_of ?? null });
                else if (kind === "room") await UpdateRoom({ id: x.id, school_id: x.school_id, name: x.name, room_type: x.room_type || "any" });
                editing = null;
                await reloadRefs();
                flash("Изменения сохранены");
        }
        // Визуальный редактор запретов: выбираем учителя/класс/кабинет и
        // кликаем по ячейкам сетки — запрет ставится/снимается сразу.
        let consPickKind = "teacher";
        let consPickId = 0;
        $: consPickList = consPickKind === "teacher" ? teachers
                : consPickKind === "class" ? classes : rooms;
        // Эффективная сущность: выбранная или первая из списка (без записи
        // обратно в состояние — иначе цикл зависимостей).
        $: consPickEntity = consPickList.find((x) => x.id === consPickId) || consPickList[0] || null;
        function setPickKind(kind) { consPickKind = kind; }
        function pickName(x) { return x.name || ""; }
        // Ограничения «недоступен», покрывающие ячейку (day, slot).
        function coveringConstraints(day, slot) {
                if (!consPickEntity) return [];
                return constraints.filter((c) =>
                        c.entity_type === consPickKind && c.entity_id === consPickEntity.id && c.is_hard &&
                        (c.type === "teacher_unavailable" || c.type === "class_unavailable" || c.type === "room_unavailable") &&
                        (c.day_of_week === null || c.day_of_week === day) &&
                        (c.timeslot_start === null || slot >= c.timeslot_start) &&
                        (c.timeslot_end === null || slot <= c.timeslot_end));
        }
        function isForbidden(day, slot) { return coveringConstraints(day, slot).length > 0; }
        async function toggleForbidden(day, slot) {
                if (!consPickEntity) { flash("Сначала добавьте учителя/класс/кабинет в справочники."); return; }
                const covering = coveringConstraints(day, slot);
                if (covering.length) {
                        for (const c of covering) await DeleteConstraint(c.id);
                        flash("Запрет снят: " + dayName(day) + " П" + (slot + 1));
                } else {
                        await CreateConstraint({
                                type: consPickKind + "_unavailable", entity_type: consPickKind,
                                entity_id: consPickEntity.id, day_of_week: day,
                                timeslot_start: slot, timeslot_end: slot,
                                is_hard: true, school_id: activeSchoolID,
                        });
                        flash("Запрещено: " + dayName(day) + " П" + (slot + 1));
                }
                await reloadRefs();
        }
        function toggleConDay(d, on) {
                con.days = on ? [...con.days, d].sort((a, b) => a - b) : con.days.filter((x) => x !== d);
        }
        // Правило «один предмет — не более N подряд» (max_consecutive_subject).
        let ruleSubject = 0, ruleMax = 2;
        async function addRunRule() {
                const n = Math.max(1, Math.min(8, ruleMax || 0));
                if (!n) { flash("Укажите число подряд (например 2)"); return; }
                await CreateConstraint({
                        school_id: activeSchoolID,
                        type: "max_consecutive_subject",
                        entity_type: ruleSubject ? "subject" : "school",
                        entity_id: ruleSubject || 0,
                        weight: n,
                        is_hard: true,
                });
                await reloadRefs();
                flash("Правило добавлено: " + (ruleSubject ? subjName(subjects, ruleSubject) : "все предметы") + " — не более " + n + " подряд");
        }

        let days = 6, slots = 8;
        // Учебные дни недели (чекбоксы в «Ограничениях»): бит 0 = Пн …
        // бит 6 = Вс. 0 — до загрузки настроек (тогда активны первые days).
        let schoolDaysMask = 0;
        const DAY_NAMES = ["Пн", "Вт", "Ср", "Чт", "Пт", "Сб", "Вс"];
        // Дни, которые рисуем в сетках/печати/PDF: включённые в маску,
        // а без маски — первые days дней (легаси-поведение).
        $: activeDayIdx = schoolDaysMask > 0
                ? DAY_NAMES.map((_, d) => d).filter((d) => schoolDaysMask & (1 << d))
                : Array.from({ length: Math.min(days, 7) }, (_, d) => d);
        let bellPeriods = [];
        let genResult = null;
	let usePrecise = false;
	// Таймаут точной генерации (сек): 0 — без лимита. Предустановки 30 / 90,
	// либо «другое» с произвольным значением.
	let solveTimeoutPreset = "none"; // "none" | "30" | "90" | "custom"
	let solveTimeoutCustom = "";
	$: solveTimeoutSec = (() => {
		if (solveTimeoutPreset === "none") return 0;
		if (solveTimeoutPreset === "custom") {
			const v = parseInt(solveTimeoutCustom, 10);
			return isNaN(v) || v < 0 ? 0 : v;
		}
		return parseInt(solveTimeoutPreset, 10);
	})();
	let isGenerating = false;
	let genID = 0;
	let pollInterval = null;
        // hasPreciseSolver is loaded once from the backend so the UI can tell the
        // user the truth: if the binary was NOT compiled with -tags ortools, the
        // "точный CP-SAT (OR-Tools)" checkbox silently runs the pure-Go fallback.
        let hasPreciseSolver = true;
        async function detectPreciseSolver() {
                try { hasPreciseSolver = await HasPreciseSolver(); }
                catch (e) { hasPreciseSolver = true; /* old binary: assume yes */ }
        }
        // "school" is the aSc-style overview: every class as a compact
        // mini-table, all on one screen. Its data kind is still "class".
        let viewMode = "school";
        $: kind = viewMode === "school" ? "class" : viewMode;
        $: overviewMode = viewMode === "school";
        // Порядок строк классов: родитель, сразу за ним его подгруппы.
        function orderClasses(list) {
                const kids = {};
                list.forEach((c) => {
                        if (c.subgroup_of) (kids[c.subgroup_of] = kids[c.subgroup_of] || []).push(c);
                });
                const out = [];
                for (const c of list) {
                        if (c.subgroup_of) continue;
                        out.push(c);
                        (kids[c.id] || []).sort((a, b) => String(a.label).localeCompare(String(b.label))).forEach((s) => out.push(s));
                }
                for (const c of list) if (!out.includes(c)) out.push(c);
                return out;
        }
$: rows = kind === "teacher"
		? teachers.map((t) => ({ id: t.id, label: t.name }))
		: kind === "room"
		? rooms.map((r) => ({ id: r.id, label: r.name }))
		: orderClasses(classes.map((c) => ({ id: c.id, label: c.name, subgroup_of: c.subgroup_of }))).filter((c) => !c.subgroup_of);

        // PDF следует выбранному «Виду» на экране — что видите, то и в файле.
        $: exportMode = viewMode;
        let pageSize = "A2";
        let orientation = "landscape";
        let compact = false;
        let pdfShowTeacher = true;
        let pdfShowRoom = true;
        let pdfWeekdaysOnly = false;
        let pdfBW = false;
        let classPage = 0;
        const classesPerPage = 10;
        $: visibleRows = overviewMode
                ? rows
                : kind === "class"
                ? rows.slice(classPage * classesPerPage, classPage * classesPerPage + classesPerPage)
                : rows;
        $: totalClassPages = Math.max(1, Math.ceil(classes.length / classesPerPage));

        // Reactive grid. THE root-cause fix for the "drag & drop does nothing
        // until you switch tabs" bug: the old template computed each cell via
        // {@const cell = cellAt(...)} inside {#each Array(days)}. Those blocks
        // only depended on `days`/`slots` — NOT on `schedule` — so after a move
        // Svelte never recomputed the cells and the grid looked frozen even
        // though the backend had already persisted the change.
        //
        // IMPORTANT Svelte detail: reactive dependencies are collected
        // SYNTACTICALLY from the statement. cellAt() reads `schedule` and
        // `conflictIDs` internally, but the compiler cannot see inside the
        // function — so we pass them as explicit arguments to make them real
        // dependencies. Any change to the schedule now rebuilds the grid.
        //
        // Индекс расписания: `${kind}|${id}|${day}|${slot}` → записи в
        // порядке schedule. Раньше cellAt/cellSubs сканировали ВЕСЬ schedule
        // на каждую ячейку (а cellSubs ещё и звал cellAt дважды на каждую
        // запись — квадрат на каждый рендер сетки). Реактивный стейтмент
        // стоит ПЕРЕД $: grid: cellAt читает scheduleIndex, и grid должен
        // пересчитаться после него.
        function cellKey(kind, id, day, slot) { return kind + "|" + id + "|" + day + "|" + slot; }
        function idxPut(idx, kind, id, d, s, en) {
                const k = cellKey(kind, id, d, s);
                const arr = idx.get(k);
                if (arr) arr.push(en); else idx.set(k, [en]);
        }
        function buildScheduleIndex(schedule) {
                const idx = new Map();
                const pos = new Map(); // id записи → позиция в schedule (порядок ячеек)
                for (let i = 0; i < schedule.length; i++) {
                        const en = schedule[i];
                        pos.set(en.id, i);
                        idxPut(idx, "class", en.class_id, en.day_of_week, en.timeslot, en);
                        idxPut(idx, "teacher", en.teacher_id, en.day_of_week, en.timeslot, en);
                        if (en.room_id) idxPut(idx, "room", en.room_id, en.day_of_week, en.timeslot, en);
                }
                return { idx, pos };
        }
        $: scheduleIndex = buildScheduleIndex(schedule);
        $: grid = buildGrid(schedule, visibleRows, kind, activeDayIdx, slots, conflictIDs);
        function buildGrid(schedule, visibleRows, viewMode, activeDayIdx, slots, conflictIDs) {
                void schedule; void conflictIDs; // (dependencies — see comment above)
                return visibleRows.map((row) => ({
                        id: row.id,
                        label: row.label,
                        cells: activeDayIdx.map((di) =>
                                Array.from({ length: slots }, (_, si) => cellAt(viewMode, row.id, di, si)))
                }));
        }

        function flash(m) { msg = m; setTimeout(() => msg = "", 3000); }

        async function loadSchools() {
                schools = (await ListSchools()) || [];
                if (schools.length && !activeSchoolID) activeSchoolID = schools[0].id;
                await loadSettings();
        }
        async function deleteSchool() {
                if (!activeSchoolID) return;
                const name = schools.find((s) => s.id === activeSchoolID)?.name || "";
                if (!await confirmAction("Удалить школу «" + name + "»? Будут удалены ВСЕ её данные: классы, учителя, предметы, кабинеты, уроки и расписание.")) return;
                await DeleteSchool(activeSchoolID);
                activeSchoolID = 0;
                await loadSchools();
                flash("Школа удалена");
        }
        async function createSchool() {
                if (!newSchoolName.trim()) { flash("Введите название школы"); return; }
                const sc = await CreateSchool(newSchoolName);
                activeSchoolID = sc.id;
                await loadSchools();
                flash("Школа создана");
        }
        async function loadSettings() {
                if (!activeSchoolID) return;
                try {
                        const raw = await GetSchoolSettings(activeSchoolID);
                        const st = raw ? JSON.parse(raw) : {};
                        if (st.days > 0) days = st.days;
                        if (st.slots > 0) slots = st.slots;
                        schoolDaysMask = st.days_mask > 0
                                ? st.days_mask
                                : (1 << Math.min(days, 7)) - 1;
                        // Звонки не сбрасываем в пустоту при несовпадении
                        // длины: дополняем/обрезаем до текущего числа уроков.
                        if (Array.isArray(st.periods)) {
                                const cur = st.periods.slice();
                                while (cur.length < slots) cur.push({ start: "", end: "" });
                                bellPeriods = cur.slice(0, slots);
                        } else {
                                bellPeriods = Array.from({ length: slots }, () => ({ start: "", end: "" }));
                        }
                } catch (e) {
                        bellPeriods = Array.from({ length: slots }, () => ({ start: "", end: "" }));
                }
        }
        // bellPeriods всегда имеет ровно slots записей: иначе bind:value
        // в таблице звонков читал несуществующий элемент и «Сохранить»
        // мог записать в базу пустые звонки.
        $: if (bellPeriods.length !== slots) {
                const cur = bellPeriods.slice();
                while (cur.length < slots) cur.push({ start: "", end: "" });
                bellPeriods = cur.slice(0, Math.max(0, slots));
        }
        async function saveSettings() {
                const periods = [];
                for (let i = 0; i < slots; i++) periods.push({ start: normalizeTime(bellPeriods[i]?.start || ""), end: normalizeTime(bellPeriods[i]?.end || "") });
                const st = { days, slots, days_mask: schoolDaysMask, periods };
                await UpdateSchoolSettings(activeSchoolID, JSON.stringify(st));
                bellPeriods = periods;
                flash("Настройки сохранены");
        }
        // Время звонков — всегда 24-часовое ЧЧ:ММ. Терпит ввод «830»,
        // «8:30», «8.30», «08:30»; пустое остаётся пустым.
        function normalizeTime(v) {
                let s = String(v || "").trim().replace(/[.,; ]+/g, ":");
                if (!s) return "";
                const digits = s.replace(/\D/g, "");
                if (!s.includes(":") && (digits.length === 3 || digits.length === 4)) {
                        s = digits.slice(0, digits.length - 2) + ":" + digits.slice(-2);
                }
                const m = s.match(/^(\d{1,2}):(\d{1,2})$/);
                if (!m) return "";
                const h = Math.min(23, +m[1]), mm = Math.min(59, +m[2]);
                return String(h).padStart(2, "0") + ":" + String(mm).padStart(2, "0");
        }
        // Переключение учебного дня (чекбокс в «Ограничениях»): последний
        // включённый день снять нельзя — расписание без дней не строится.
        // Меняется сразу везде: сетка, генерация, печать и PDF.
        function toggleSchoolDay(d, on) {
                const next = on ? (schoolDaysMask | (1 << d)) : (schoolDaysMask & ~(1 << d));
                if (next === 0) { flash("Хотя бы один учебный день должен быть включён"); return; }
                schoolDaysMask = next;
                if (activeSchoolID) {
                        const periods = [];
                        for (let i = 0; i < slots; i++) periods.push({ start: normalizeTime(bellPeriods[i]?.start || ""), end: normalizeTime(bellPeriods[i]?.end || "") });
                        UpdateSchoolSettings(activeSchoolID, JSON.stringify({ days, slots, days_mask: schoolDaysMask, periods })).catch(() => {});
                }
        }
        function onSlotsChange() {
                const cur = bellPeriods.slice();
                while (cur.length < slots) cur.push({ start: "", end: "" });
                bellPeriods = cur.slice(0, slots);
        }
        async function reloadRefs() {
                if (!activeSchoolID) return;
                const res = await Promise.all([
                        ListTeachers(activeSchoolID), ListSubjects(activeSchoolID),
                        ListClasses(activeSchoolID), ListRooms(activeSchoolID),
                        ListLessons(activeSchoolID), ListConstraints(activeSchoolID)
                ]);
                [teachers, subjects, classes, rooms, lessons, constraints] = res.map((x) => x || []);
        }
        async function reloadSchedule() {
                if (!activeSchoolID) return;
                schedule = (await ListSchedule(activeSchoolID)) || [];
                // conflictIDs пересчитывается реактивно из schedule+constraints
        }

        let conflictIDs = new Set();
        // Красным помечаем и двойные брони, и уроки, вставшие на запрещённую
        // ячейку жёсткого ограничения «недоступен» (учитель/класс/кабинет).
        $: conflictIDs = computeConflictIDs(schedule, constraints);
        function violatesUnavailable(c, e) {
                if (c.entity_type === "teacher" ? e.teacher_id !== c.entity_id
                        : c.entity_type === "class" ? e.class_id !== c.entity_id
                        : e.room_id !== c.entity_id) return false;
                if (c.day_of_week != null && e.day_of_week !== c.day_of_week) return false;
                if (c.timeslot_start != null && e.timeslot < c.timeslot_start) return false;
                if (c.timeslot_end != null && e.timeslot > c.timeslot_end) return false;
                return true;
        }
        function computeConflictIDs(schedule, constraints) {
                const ids = new Set();
                // Учитель/кабинет: группируем по (значение, день, слот).
                const maps = { teacher_id: {}, class_id: {}, room_id: {} };
                for (const e of schedule) {
                        const k = e.day_of_week * 1000 + e.timeslot;
                        const tv = e.teacher_id, rv = e.room_id;
                        if (tv) {
                                if (!maps.teacher_id[tv]) maps.teacher_id[tv] = {};
                                if (!maps.teacher_id[tv][k]) maps.teacher_id[tv][k] = [];
                                maps.teacher_id[tv][k].push(e.id);
                        }
                        if (rv) {
                                if (!maps.room_id[rv]) maps.room_id[rv] = {};
                                if (!maps.room_id[rv][k]) maps.room_id[rv][k] = [];
                                maps.room_id[rv][k].push(e.id);
                        }
                        // Класс: пары уроков в одном слоте — конфликт, если
                        // тела пересекаются (целый класс vs его подгруппа);
                        // две подгруппы одного родителя disjoint — можно параллельно.
                        if (!maps.class_id[e.class_id]) maps.class_id[e.class_id] = {};
                        if (!maps.class_id[e.class_id][k]) maps.class_id[e.class_id][k] = [];
                        maps.class_id[e.class_id][k].push(e.id);
                }
                for (const f of ["teacher_id"]) {
                        for (const v in maps[f]) {
                                for (const k in maps[f][v]) {
                                        if (maps[f][v][k].length > 1) maps[f][v][k].forEach((id) => ids.add(id));
                                }
                        }
                }
                // Кабинет: попарно; уроки одной семьи класса (родитель +
                // его подгруппы) могут делить кабинет — половины класса вмещаются
                for (const rv in maps.room_id) {
                        const arr = maps.room_id[rv];
                        for (let i = 0; i < arr.length; i++) {
                                for (let j = i + 1; j < arr.length; j++) {
                                        const a = arr[i], b = arr[j];
                                        const famA = classBody(a.class_id), famB = classBody(b.class_id);
                                        const sameFamily = famA.includes(b.class_id) || famB.includes(a.class_id);
                                        if (!sameFamily) {
                                                ids.add(a.id); ids.add(b.id);
                                        }
                                }
                        }
                }
                // Классы: дубль = один и тот же класс дважды в слоте.
                // Родитель + подгруппа параллельно — законно (деление класса).
                for (const cv in maps.class_id) {
                        for (const k in maps.class_id[cv]) {
                                if (maps.class_id[cv][k].length > 1) maps.class_id[cv][k].forEach((id) => ids.add(id));
                        }
                }
                for (const c of constraints) {
                        if (!c.is_hard) continue;
                        if (c.type !== "teacher_unavailable" && c.type !== "class_unavailable" && c.type !== "room_unavailable") continue;
                        for (const e of schedule) if (violatesUnavailable(c, e)) ids.add(e.id);
                }
                // Делёные уроки: запись половинки, у которой в этом же слоте
                // нет сестринской записи, стоит не параллельно — конфликт.
                const twinOfLesson = {};
                for (const [lid, t] of dividedTwinOf) twinOfLesson[lid] = t.id;
                const halfAt = {};
                for (const e of schedule) {
                        if (!twinOfLesson[e.lesson_id]) continue;
                        const k = e.lesson_id + ":" + e.day_of_week + ":" + e.timeslot;
                        halfAt[k] = (halfAt[k] || 0) + 1;
                }
                for (const e of schedule) {
                        const twinId = twinOfLesson[e.lesson_id];
                        if (!twinId) continue;
                        if (!halfAt[twinId + ":" + e.day_of_week + ":" + e.timeslot]) ids.add(e.id);
                }
                return ids;
        }

        let report = { conflicts: [], unplaced: [], violations: [], classStats: [], reasonGroups: [], totalHours: 0, placedHours: 0, missingHours: 0, placedPct: 100 };
        function computeConflictReport() {
                // Та же логика: учитель/кабинет по значению, классы —
                // попарным пересечением тел (целый класс vs подгруппа).
                const byKey = { teacher_id: {}, class_id: {}, room_id: {} };
                for (const e of schedule) {
                        const key = e.day_of_week * 1000 + e.timeslot;
                        for (const f of ["teacher_id", "class_id", "room_id"]) {
                                const v = e[f];
                                if (!v) continue;
                                (byKey[f][v + ":" + key] = byKey[f][v + ":" + key] || []).push(e);
                        }
                }
                const conflicts = [];
                // Учитель и кабинет не могут вести/занимать два урока сразу.
                // Для класса конфликтуют только повторные записи того же класса:
                // родитель и его подгруппа могут заниматься параллельно.
                const conflictTypes = [
                        ["teacher_id", "Учитель", (e) => teachName(teachers, e.teacher_id)],
                        ["room_id", "Кабинет", (e) => roomName(rooms, e.room_id)],
                        ["class_id", "Класс", (e) => className(classes, e.class_id)],
                ];
                for (const [field, type, whoFor] of conflictTypes) {
                        for (const entries of Object.values(byKey[field])) {
                                let conflicting = entries;
                                if (field === "room_id") {
                                        // Параллельные половины одного класса могут делить кабинет.
                                        const involved = new Set();
                                        for (let i = 0; i < entries.length; i++) {
                                                for (let j = i + 1; j < entries.length; j++) {
                                                        const a = entries[i], b = entries[j];
                                                        const bodyA = classBody(a.class_id), bodyB = classBody(b.class_id);
                                                        if (!bodyA.includes(b.class_id) && !bodyB.includes(a.class_id)) {
                                                                involved.add(a); involved.add(b);
                                                        }
                                                }
                                        }
                                        conflicting = [...involved];
                                }
                                if (conflicting.length < 2) continue;
                                const first = conflicting[0];
                                conflicts.push({
                                        type,
                                        day: first.day_of_week,
                                        slot: first.timeslot,
                                        items: conflicting.map((e) => ({
                                                subject: subjName(subjects, e.subject_id),
                                                who: whoFor(e),
                                        })),
                                });
                        }
                }
                const placedByLesson = {};
                for (const e of schedule) placedByLesson[e.lesson_id] = (placedByLesson[e.lesson_id] || 0) + 1;
                const teacherHours = {};
                for (const e of schedule) teacherHours[e.teacher_id] = (teacherHours[e.teacher_id] || 0) + 1;
                const totalCells = activeDayIdx.length * slots;

                // Занятость по ячейкам: «ид:день:слот» в Set (O(1)-проверка).
                const teacherAt = new Set(), classAt = new Set(), roomAt = new Set();
                // Нагрузка класса с учётом подгрупп (подгруппа суммируется в родителя).
                const classFamilyPlaced = {};
                const familyOf = {};
                for (const c of classes) familyOf[c.id] = c.subgroup_of ? c.subgroup_of : c.id;
                for (const e of schedule) {
                        const k = e.day_of_week + ":" + e.timeslot;
                        if (e.teacher_id) teacherAt.add(e.teacher_id + ":" + k);
                        classAt.add(e.class_id + ":" + k);
                        if (e.room_id) roomAt.add(e.room_id + ":" + k);
                        const fam = familyOf[e.class_id] ?? e.class_id;
                        classFamilyPlaced[fam] = (classFamilyPlaced[fam] || 0) + 1;
                }
                // Ячейки, запрещённые жёсткими ограничениями «недоступен», по сущности (кешируем).
                const bannedCache = new Map();
                function bannedCells(entityType, entityId) {
                        const key = entityType + ":" + entityId;
                        if (bannedCache.has(key)) return bannedCache.get(key);
                        const cells = new Set();
                        for (const c of constraints) {
                                if (!c.is_hard) continue;
                                if (c.type !== entityType + "_unavailable") continue;
                                if (c.entity_type !== entityType || c.entity_id !== entityId) continue;
                                const affectedDays = c.day_of_week != null ? [c.day_of_week] : activeDayIdx;
                                const start = c.timeslot_start ?? 0;
                                const end = c.timeslot_end ?? (slots - 1);
                                for (const d of affectedDays) for (let s = start; s <= end; s++) cells.add(d + ":" + s);
                        }
                        bannedCache.set(key, cells);
                        return cells;
                }
                // Классы, делящие с данным классом одну ячейку (родитель ↔ его подгруппы) —
                // та же семантика, что buildClassSets в солвере.
                function classBody(classId) {
                        const out = [classId];
                        for (const kid of (kidsOf.get(classId) || [])) out.push(kid);
                        const c = classById.get(classId);
                        if (c && c.subgroup_of) out.push(c.subgroup_of);
                        return out;
                }
                // Кабинеты, допустимые для предмета — зеркало allowedRooms() в солвере.
                function allowedRoomIds(subj) {
                        const rt = subj && subj.requires_room_type ? subj.requires_room_type : "any";
                        let out = rooms.filter((r) => rt === "any" || r.room_type === "any" || r.room_type === rt);
                        if (out.length === 0 && rooms.length > 0) out = rooms.slice();
                        return out;
                }
                // Диагностика как в aSc/FET: проходим ВСЕ ячейки недели и считаем,
                // чем занята каждая — чтобы сказать не «не размещено», а ПОЧЕМУ.
                function unplacedDiagnose(l) {
                        const subj = subjectById.get(l.subject_id);
                        const body = classBody(l.class_id);
                        const allowed = allowedRoomIds(subj);
                        const bt = bannedCells("teacher", l.teacher_id);
                        const bcs = body.map((b) => bannedCells("class", b));
                        let free = 0, busyT = 0, busyC = 0, busyR = 0, banned = 0;
                        for (const d of activeDayIdx) {
                                for (let s = 0; s < slots; s++) {
                                        const k = d + ":" + s;
                                        if (teacherAt.has(l.teacher_id + ":" + k)) { busyT++; continue; }
                                        if (body.some((b) => classAt.has(b + ":" + k))) { busyC++; continue; }
                                        if (bt.has(k)) { banned++; continue; }
                                        if (bcs.some((set) => set.has(k))) { banned++; continue; }
                                        if (!allowed.some((r) => !roomAt.has(r.id + ":" + k))) { busyR++; continue; }
                                        free++;
                                }
                        }
                        return { free, busyT, busyC, busyR, banned, total: totalCells, subj };
                }
                // Причина неразмещения урока — конкретная, с числами по ячейкам.
                function unplacedReason(l) {
                        const dg = unplacedDiagnose(l);
                        const t = teacherById.get(l.teacher_id);
                        const c = classById.get(l.class_id);
                        if (rooms.length === 0) return "В школе не задано ни одного кабинета — размещать некуда";

                        // Делёная пара: половинки ходят только вместе, одна без другой не встаёт.
                        const twin = dividedTwinOf.get(l.id);
                        if (twin && (placedByLesson[twin.id] || 0) < (twin.hours_per_week || 0)) {
                                const twinCls = classById.get(twin.class_id);
                                return `Делёный урок: парная половина для ${twinCls ? twinCls.name : "?"} не размещена — половинки ходят только парой`;
                        }

                        if (dg.free > 0) {
                                return `Есть ${dg.free} свободных окон из ${dg.total}, но солвер не успел их занять за отведённое время`;
                        }
                        // Свободных окон нет — разбираем, чем занята неделя.
                        const parts = [];
                        if (dg.busyT) parts.push(`учитель занят в ${dg.busyT}`);
                        if (dg.busyC) parts.push(`класс занят в ${dg.busyC}`);
                        if (dg.busyR) parts.push(`нет свободного кабинета в ${dg.busyR}`);
                        if (dg.banned) parts.push(`запрещено ограничениями в ${dg.banned}`);
                        const detail = parts.length ? " (" + parts.join(", ") + ")" : "";
                        const top = Math.max(dg.busyT, dg.busyC, dg.busyR, dg.banned);
                        let head = `Свободных окон нет ни в одной из ${dg.total} ячеек`;
                        if (top === dg.busyT) head = `Учитель ${t ? t.name : "?"} занят во всех ${dg.total} ячейках`;
                        else if (top === dg.busyC) head = `Класс ${c ? c.name : "?"} занят во всех ${dg.total} ячейках`;
                        else if (top === dg.busyR) head = `Нет свободного кабинета ни в одну из ${dg.total} ячеек`;
                        else if (top === dg.banned) head = `Все ${dg.total} ячеек закрыты ограничениями «недоступен»`;
                        return head + detail;
                }
                const unplaced = lessons.filter((l) => (placedByLesson[l.id] || 0) < (l.hours_per_week || 0))
                        .map((l) => ({ subject: subjName(subjects, l.subject_id), cls: className(classes, l.class_id), need: l.hours_per_week, got: placedByLesson[l.id] || 0, reason: unplacedReason(l) }));

                // ---- Сводка для панели расстановки ----
                const totalHours = lessons.reduce((n, l) => n + (l.hours_per_week || 0), 0);
                const placedHours = schedule.length;
                const missingHours = Math.max(0, totalHours - placedHours);
                const placedPct = totalHours ? Math.round((placedHours / totalHours) * 100) : 100;
                const classNeeds = {};
                for (const l of lessons) {
                        const fam = familyOf[l.class_id] ?? l.class_id;
                        classNeeds[fam] = (classNeeds[fam] || 0) + (l.hours_per_week || 0);
                }
                const classStats = Object.keys(classNeeds).map((id) => {
                        const cid = Number(id);
                        const need = classNeeds[id];
                        const placed = classFamilyPlaced[cid] || 0;
                        return { name: className(classes, cid), need, placed, missing: Math.max(0, need - placed) };
                }).sort((a, b) => b.missing - a.missing || a.name.localeCompare(b.name));
                // Одинаковые причины сворачиваем в один пункт со счётчиком.
                const reasonGroups = [];
                for (const u of unplaced) {
                        let g = reasonGroups.find((x) => x.reason === u.reason);
                        if (!g) { g = { reason: u.reason, items: [] }; reasonGroups.push(g); }
                        g.items.push(u);
                }
                reasonGroups.sort((a, b) => b.items.length - a.items.length);
                const teacherStats = teachers.filter((t) => teacherHours[t.id] || 0)
                        .map((t) => ({ name: t.name, placed: teacherHours[t.id] || 0 }))
                        .sort((a, b) => b.placed - a.placed || a.name.localeCompare(b.name));
                const violations = [];
                for (const c of constraints) {
                        if (!c.is_hard) continue;
                        if (c.type !== "teacher_unavailable" && c.type !== "class_unavailable" && c.type !== "room_unavailable") continue;
                        for (const e of schedule) {
                                if (violatesUnavailable(c, e)) {
                                        violations.push({ day: e.day_of_week, slot: e.timeslot, what: constraintTypeLabel(c.type) + " · " + constraintEntityLabel(c) });
                                }
                        }
                }
                // Делёные уроки: половинки без пары в своём слоте.
                for (const e of schedule) {
                        const twinId = dividedTwinOf.get(e.lesson_id)?.id || 0;
                        if (!twinId) continue;
                        const hasTwin = schedule.some((x) => x.lesson_id === twinId
                                && x.day_of_week === e.day_of_week && x.timeslot === e.timeslot);
                        if (!hasTwin) {
                                conflicts.push({
                                        type: "Деление",
                                        day: e.day_of_week, slot: e.timeslot,
                                        items: [{ subject: subjName(subjects, e.subject_id), who: className(classes, e.class_id) + " — половинка без пары" }]
                                });
                        }
                }
                return { conflicts, unplaced, violations, classStats, teacherStats, reasonGroups, totalHours, placedHours, missingHours, placedPct };
        }
        // Same syntactic-dependency rule as buildGrid above: pass the state
        // this report is derived from as explicit arguments.
        $: report = computeConflictReport(schedule, lessons, subjects, teachers, classes, rooms, constraints, slots, activeDayIdx);

        let history = [];
        async function pushHistory() {
                history.push(JSON.parse(JSON.stringify(schedule)));
                if (history.length > 50) history.shift();
        }
        async function undo() {
                if (!history.length) return;
                const prev = history.pop();
                await ReplaceSchedule(activeSchoolID, prev);
                await reloadSchedule();
                flash("Отменено");
        }

        $: if (activeSchoolID) { reloadRefs(); reloadSchedule(); }

        async function addTeacher() {
                if (!t.name.trim()) { flash("Введите имя учителя"); return; }
                await CreateTeacher({ ...t, school_id: activeSchoolID });
                t = { name: "", short_name: "" };
                await reloadRefs();
        }
        async function addSubject() {
                if (!s.name.trim()) { flash("Введите название предмета"); return; }
                await CreateSubject({ ...s, school_id: activeSchoolID });
                s = { name: "", short_name: "", requires_room_type: "any" };
                await reloadRefs();
        }
        async function addClass() {
                if (!c.name.trim()) { flash("Введите название класса"); return; }
                await CreateClass({ ...c, school_id: activeSchoolID });
                c = { name: "", room_id: 0, subgroup_of: null };
                await reloadRefs();
        }
        async function addRoom() {
                if (!r.name.trim()) { flash("Введите название кабинета"); return; }
                await CreateRoom({ ...r, school_id: activeSchoolID });
                r = { name: "", room_type: "any" };
                await reloadRefs();
        }
        async function addLesson() {
                if (!l.class_id || !l.subject_id || !l.teacher_id) { flash("Выберите класс, предмет и учителя"); return; }
                const hours = Math.max(1, Math.min(40, l.hours_per_week || 1));
                await CreateLesson({ ...l, school_id: activeSchoolID, hours_per_week: hours });
                l = { class_id: 0, subject_id: 0, teacher_id: 0, hours_per_week: 1 };
                await reloadRefs();
        }
        async function addLessonForClass() {
                if (!curClass || !l.subject_id || !l.teacher_id) { flash("Выберите класс, предмет и учителя"); return; }
                await CreateLesson({ school_id: activeSchoolID, class_id: curClass, subject_id: l.subject_id, teacher_id: l.teacher_id, hours_per_week: l.hours_per_week || 1 });
                l = { class_id: 0, subject_id: 0, teacher_id: 0, hours_per_week: 1 };
                await reloadRefs();
        }
        async function updateLesson(x) {
                try {
                        await UpdateLesson({ id: x.id, school_id: x.school_id, class_id: x.class_id, subject_id: x.subject_id, teacher_id: x.teacher_id, hours_per_week: x.hours_per_week || 1 });
                } catch (e) { flash("Ошибка обновления урока: " + (e && e.message ? e.message : e)); }
        }
        async function removeLesson(id) { if (!await confirmAction("Удалить урок?")) return; await pushHistory(); await DeleteLesson(id); await reloadRefs(); flash("Урок удалён"); }
        // window.confirm не работает в Wails (WebView2 гасит JS-диалоги),
        // поэтому подтверждение — собственное модальное окно, одинаковое
        // и в браузерном, и в десктопном режиме.
        let confirmBox = null; // { message, confirmLabel }
        let confirmResolve = null;
        function confirmAction(message, confirmLabel = "Удалить") {
                return new Promise((resolve) => {
                        confirmBox = { message, confirmLabel };
                        confirmResolve = resolve;
                });
        }
        function settleConfirm(answer) {
                if (confirmResolve) confirmResolve(answer);
                confirmBox = null;
                confirmResolve = null;
        }
        async function removeTeacher(id) { if (!await confirmAction("Удалить учителя? Все его уроки тоже будут удалены.")) return; await pushHistory(); await DeleteTeacher(id); await reloadRefs(); flash("Учитель удалён"); }
        async function removeSubject(id) { if (!await confirmAction("Удалить предмет? Связанные уроки тоже будут удалены.")) return; await pushHistory(); await DeleteSubject(id); await reloadRefs(); flash("Предмет удалён"); }
        async function removeClass(id) { if (!await confirmAction("Удалить класс? Все уроки и расписание класса будут удалены.")) return; await pushHistory(); await DeleteClass(id); await reloadRefs(); flash("Класс удалён"); }
        async function removeRoom(id) { if (!await confirmAction("Удалить кабинет? Связанные ячейки расписания тоже будут удалены.")) return; await pushHistory(); await DeleteRoom(id); await reloadRefs(); await reloadSchedule(); flash("Кабинет удалён"); }
        async function removeConstraint(id) { await DeleteConstraint(id); await reloadRefs(); flash("Ограничение удалено"); }
        async function removeEntry(id) { await pushHistory(); await DeleteScheduleEntry(id); await reloadSchedule(); }

	async function generate() {
			if (!lessons.length) { flash("Нет уроков в учебном плане — добавьте их на вкладке «Уроки»."); return; }
			const occurrences = lessons.reduce((a, l) => a + (l.hours_per_week || 0), 0);
			if (!usePrecise && occurrences > 200) {
					usePrecise = true;
					if (hasPreciseSolver) {
							flash("Крупная школа: включён точный CP-SAT (OR-Tools).");
					} else {
							flash("Крупная школа: OR-Tools недоступен в этой сборке — используется быстрый эвристический решатель (может не разместить все уроки).");
					}
			}
			if (usePrecise) {
					return await generatePrecise();
			}
			await pushHistory();
			try {
					isGenerating = true;
					const t0 = performance.now();
					genResult = await Generate(activeSchoolID, days, slots, schoolDaysMask) || {};
					await reloadSchedule();
					isGenerating = false;
					const dt = Math.round(performance.now() - t0);
					flash(`Размещено ${genResult.placed}/${genResult.total}, нарушений (мягких): ${genResult.violations} (${dt} с)`);
			} catch (e) {
					isGenerating = false;
					flash("Ошибка генерации: " + (e && e.message ? e.message : e));
			}
	}
	async function generatePrecise() {
			if (!lessons.length) { flash("Нет уроков в учебном плане — добавьте их на вкладке «Уроки»."); return; }
			await pushHistory();
			try {
					isGenerating = true;
					genID = await StartPreciseGeneration(activeSchoolID, days, slots, schoolDaysMask, solveTimeoutSec);
					pollInterval = setInterval(async () => {
							const p = await GetGenerationProgress(genID);
							if (p.Status === "done" || p.Status === "error") {
									clearInterval(pollInterval);
									pollInterval = null;
									isGenerating = false;
									await reloadSchedule();
									if (p.Status === "error") {
											flash("Ошибка генерации: " + (p.Error || "неизвестная ошибка"));
											return;
									}
									genResult = { placed: p.Placed, total: p.Total, violations: 0, elapsed_ms: 0 };
									flash(`CP-SAT: размещено ${p.Placed}/${p.Total}`);
							} else {
									genResult = { placed: p.Placed, total: p.Total, violations: 0, elapsed_ms: 0 };
							}
					}, 500);
			} catch (e) {
					isGenerating = false;
					if (pollInterval) { clearInterval(pollInterval); pollInterval = null; }
					flash("Ошибка генерации: " + (e && e.message ? e.message : e));
			}
	}
        // applyMove performs a true optimistic move/swap.
        //
        // Old behavior: await backend → update local schedule → reload from
        // backend. That meant the cell visually "froze" for the whole Wails +
        // SQLite round trip and, if the user navigated away during that window,
        // the optimistic update never rendered — so they had to switch tabs to
        // see the change.
        //
        // New behavior:
        //   1. Snapshot the current schedule (for rollback) and update the
        //      local `schedule` array IMMEDIATELY so the grid re-renders on the
        //      very next microtask, before any IPC.
        //   2. THEN call the backend. The DB is the source of truth, so on the
        //      error path we restore the snapshot so the UI matches the DB.
        //   3. On success we DO NOT call reloadSchedule() — the optimistic
        //      state already matches what the DB just persisted, so a fresh
        //      round trip would only waste time and could momentarily flicker
        //      the table. This is the key fix for the "need to navigate away
        //      and back" symptom.
        async function applyMove(id, kind, rowId, day, slot) {
                const src = schedule.find(en => en.id === id);
                if (!src) {
                        flash("⚠ Не найден исходный урок — возможно, расписание изменилось. Обновите вкладку.");
                        return;
                }
                // Source's own row identifier (its class/teacher/room id).
                const srcRowId = kind === "class" ? src.class_id : kind === "teacher" ? src.teacher_id : src.room_id;
                // Подгруппа может двигаться в строке своего родительского класса.
                const srcParent = kind === "class" ? subParentOf(srcRowId) : null;
                if (srcRowId !== rowId && srcParent !== rowId) {
                        const rowKindLabel = kind === "class" ? "классами" : kind === "teacher" ? "учителями" : "кабинетами";
                        flash(`⚠ Нельзя перемещать урок между ${rowKindLabel} (это нарушило бы структуру расписания). Только в пределах одной строки.`);
                        return;
                }
                // Find target in same row at the destination cell.
                // Ячейка класса вмещает ДО ДВУХ уроков: целоклассовый +
                // подгруппа, или две подгруппы параллельно (деление класса).
                const target = schedule.find(en => {
                        if (en.id === id) return false;
                        if (en.day_of_week !== day || en.timeslot !== slot) return false;
                        if (kind === "class") return en.class_id === rowId || subParentOf(en.class_id) === rowId;
                        if (kind === "teacher") return en.teacher_id === rowId;
                        return en.room_id === rowId;
                });
                const cellCount = schedule.filter(en => en.id !== id && en.day_of_week === day && en.timeslot === slot &&
                        (en.class_id === rowId || subParentOf(en.class_id) === rowId)).length;
                if (kind === "class" && cellCount >= 2) {
                        flash("⚠ Ячейка заполнена: максимум два урока (класс + подгруппа).");
                        return;
                }

                // Делёный урок (модель aSc): половинки одного предмета у
                // подгрупп одного родителя перемещаются ПАРОЙ, иначе
                // параллельность ломается (половинка остаётся без пары).
                const twinLesson = kind === "class" ? dividedTwinLesson(src.lesson_id) : null;
                if (kind !== "class" && twinLesson) {
                        flash("⚠ Делёный урок перемещается парой — двигайте его в сетке класса.");
                        return;
                }
                if (twinLesson) {
                        const twinEntry = schedule.find((en) => en.lesson_id === twinLesson.id
                                && en.day_of_week === src.day_of_week && en.timeslot === src.timeslot);
                        if (twinEntry) {
                                const famCount = schedule.filter((en) => en.id !== src.id
                                        && en.day_of_week === day && en.timeslot === slot
                                        && (en.class_id === rowId || subParentOf(en.class_id) === rowId)).length;
                                if (famCount > 0) {
                                        flash("⚠ Делёный урок перемещается парой — целевая ячейка должна быть пустой.");
                                        return;
                                }
                                const snapshot = schedule.slice();
                                await pushHistory();
                                schedule = schedule.map((en) => (en.id === src.id || en.id === twinEntry.id)
                                        ? { ...en, day_of_week: day, timeslot: slot }
                                        : en);
                                flash(`✓ Делёный урок перемещён парой → ${dayName(day)} П${slot + 1}`);
                                try {
										await MoveEntries(activeSchoolID, [src.id, twinEntry.id], day, slot);
                                } catch (err) {
                                        schedule = snapshot;
                                        flash(`⚠ Не удалось переместить пару: ${err && err.message ? err.message : err}`);
                                        return;
                                }
                                try { schedule = (await ListSchedule(activeSchoolID)) || []; } catch (e) { /* optimistic state */ }
                                return;
                        }
                        // Пары в исходном слоте нет (состояние уже разъединено,
                        // подсвечено красным) — одиночный перенос позволяет
                        // «приклеить» половинку обратно к её паре.
                }

                // Snapshot for rollback if the backend rejects the change.
                const snapshot = schedule.slice();
                await pushHistory();

                // 1. Optimistic UI update — fires reactivity immediately, before
                //    any IPC. This is the fix for the "drag does nothing until
                //    you switch tabs" bug.
                if (target) {
                        schedule = schedule.map((en) => en.id === src.id
                                ? { ...en, day_of_week: day, timeslot: slot }
                                : en.id === target.id
                                ? { ...en, day_of_week: src.day_of_week, timeslot: src.timeslot }
                                : en);
                        // conflictIDs пересчитывается реактивно из schedule+constraints
                        flash(`✓ Поменяли местами: ${cellLabelShort(src)} (${dayName(src.day_of_week)} П${src.timeslot + 1}) ⟷ ${cellLabelShort(target)} (${dayName(day)} П${slot + 1})`);
                } else {
                        schedule = schedule.map((en) => en.id === id
                                ? { ...en, day_of_week: day, timeslot: slot }
                                : en);
                        // conflictIDs пересчитывается реактивно из schedule+constraints
                        flash(`✓ Перемещено: ${cellLabelShort(src)} → ${dayName(day)} П${slot + 1}`);
                }

                // 2. Persist to the backend. On error, roll back to the
                //    pre-move state so the grid matches the DB again.
                try {
                        if (target) {
                                await SwapEntries(activeSchoolID, src.id, day, slot, target.id, src.day_of_week, src.timeslot);
                        } else {
                                await MoveEntry(activeSchoolID, id, day, slot);
                        }
                } catch (err) {
                        // Restore the snapshot and let the user know the move
                        // was rejected (e.g. DB constraint, lost race with
                        // another edit). Do NOT silently overwrite — the user
                        // must see their move was rolled back.
                        schedule = snapshot;
                        // conflictIDs пересчитывается реактивно из schedule+constraints
                        flash(`⚠ Не удалось переместить: ${err && err.message ? err.message : err}`);
                        return;
                }
                // 3. Success. The optimistic state already matches what the DB
                //    just persisted, but we still quietly re-read the schedule
                //    once so the UI is guaranteed to equal the DB (covers any
                //    serialization nuance). The grid is fully reactive now, so
                //    this refresh renders instantly instead of "freezing" like
                //    the old template did.
                try {
                        schedule = (await ListSchedule(activeSchoolID)) || [];
                        // conflictIDs пересчитывается реактивно из schedule+constraints
                } catch (e) { /* keep the optimistic state on refresh failure */ }
        }
        // ------------------------------------------------------------------
        // Drag & Drop engine (v2).
        //
        // Must work inside the Wails webview (WebView2 / WKWebView) with
        // mouse AND touch/pen. Key properties:
        //   * setPointerCapture on the pressed cell → every later pointer
        //     event is delivered to us no matter where the pointer goes;
        //   * hit-testing is done with getBoundingClientRect scans, never
        //     elementFromPoint (unreliable during pointer capture);
        //   * the cell under the pointer is highlighted (drop-target) so the
        //     user SEES where the lesson will land;
        //   * movement < 6px counts as a click (select / place-to-cell);
        //   * Escape cancels a drag in progress;
        //   * near the viewport edges we auto-scroll the grid.
        // ------------------------------------------------------------------
        let selectedEntry = null;
        let pendingDrag = null;   // { id, kind, rowId, day, slot, label, pointerId }
        let startPos = null;      // pointer position where the drag started
        let dragging = false;
        let ghost = null;         // { label, x, y } — floating label under the cursor
        let dropTarget = null;    // { rowId, day, slot } — highlighted destination cell
        let lastPointer = { x: 0, y: 0 };
        let dropCheckPending = false;

        // Engine-independent hit-test: find the schedule cell whose bounding
        // rect contains the pointer.
        function hitCell(x, y) {
                const tds = document.querySelectorAll("td[data-cell]");
                for (let i = 0; i < tds.length; i++) {
                        const r = tds[i].getBoundingClientRect();
                        if (x >= r.left && x <= r.right && y >= r.top && y <= r.bottom) return tds[i];
                }
                return null;
        }

        function readCellAttrs(td) {
                const rowId = parseInt(td.getAttribute("data-row"));
                const day = parseInt(td.getAttribute("data-day"));
                const slot = parseInt(td.getAttribute("data-slot"));
                if (isNaN(rowId) || isNaN(day) || isNaN(slot)) return null;
                return { rowId, day, slot };
        }

        function setDropTargetFromPoint(x, y) {
                if (!pendingDrag) { dropTarget = null; return; }
                const td = hitCell(x, y);
                const dst = td ? readCellAttrs(td) : null;
                // Never highlight the source cell itself.
                if (dst && dst.rowId === pendingDrag.rowId && dst.day === pendingDrag.day && dst.slot === pendingDrag.slot) {
                        dropTarget = null;
                        return;
                }
                dropTarget = dst;
        }

        // Throttle hit-testing to one scan per animation frame.
        function scheduleDropCheck() {
                if (dropCheckPending) return;
                dropCheckPending = true;
                requestAnimationFrame(() => {
                        dropCheckPending = false;
                        setDropTargetFromPoint(lastPointer.x, lastPointer.y);
                });
        }

        // Auto-scroll window (vertical) and the grid container (horizontal)
        // while dragging near an edge. The loop self-stops when dragging ends.
        function autoScrollTick() {
                if (!dragging) return;
                const m = 48, sp = 14;
                if (lastPointer.y < m) window.scrollBy(0, -sp);
                else if (lastPointer.y > window.innerHeight - m) window.scrollBy(0, sp);
                const sc = document.querySelector(".grid-scroll");
                if (sc) {
                        const r = sc.getBoundingClientRect();
                        if (lastPointer.x > r.left && lastPointer.x < r.right) {
                                if (lastPointer.x < r.left + m) sc.scrollLeft -= sp;
                                else if (lastPointer.x > r.right - m) sc.scrollLeft += sp;
                        }
                }
                requestAnimationFrame(autoScrollTick);
        }

        function onPointerDown(e, cell, kind, rowId, day, slot) {
                if (e.pointerType === "mouse" && e.button !== 0) return;
                if (e.target && e.target.closest && e.target.closest(".cell-x")) return;
                e.preventDefault();
                pendingDrag = { id: cell ? cell.id : null, kind, rowId, day, slot, label: cell ? cell.label : "", pointerId: e.pointerId };
                dragging = false;
                ghost = null;
                dropTarget = null;
                startPos = { x: e.clientX, y: e.clientY };
                lastPointer = { x: e.clientX, y: e.clientY };
                // Capture the pointer on the pressed cell: guarantees that
                // pointermove/pointerup keep coming even when the cursor
                // leaves the <td> (this is the piece the previous attempt was
                // missing — without it some drags silently died mid-move).
                try {
                        if (e.pointerId !== undefined && e.target && e.target.setPointerCapture) {
                                e.target.setPointerCapture(e.pointerId);
                        }
                } catch (err) { /* non-fatal */ }
        }
        function onPointerMove(e) {
                if (!pendingDrag) return;
                if (pendingDrag.pointerId !== undefined && e.pointerId !== undefined && e.pointerId !== pendingDrag.pointerId) return;
                lastPointer = { x: e.clientX, y: e.clientY };
                const dx = e.clientX - startPos.x, dy = e.clientY - startPos.y;
                if (!dragging && Math.hypot(dx, dy) > 6) {
                        dragging = true;
                        requestAnimationFrame(autoScrollTick);
                }
                if (dragging) {
                        ghost = { label: pendingDrag.label, x: e.clientX, y: e.clientY };
                        scheduleDropCheck();
                }
        }
        async function onPointerUp(e) {
                if (!pendingDrag) { dragging = false; ghost = null; dropTarget = null; return; }
                const pd = pendingDrag;
                const wasDrag = dragging;
                const dst = dropTarget;
                pendingDrag = null;
                dragging = false;
                ghost = null;
                dropTarget = null;
                if (wasDrag && pd.id) {
                        // Prefer the highlighted target; fall back to a fresh
                        // hit-test at the release point.
                        let target = dst;
                        if (!target && e.clientX !== undefined) {
                                const td = hitCell(e.clientX, e.clientY);
                                target = td ? readCellAttrs(td) : null;
                        }
                        if (target) {
                                if (target.day === pd.day && target.slot === pd.slot && target.rowId === pd.rowId) return;
                                try { await applyMove(pd.id, pd.kind, target.rowId, target.day, target.slot); }
                                catch (err) { flash("Ошибка: " + (err && err.message ? err.message : err)); }
                        }
                        return;
                }
                // Click without drag: select or click-to-move
                if (!pd.id) {
                        if (selectedEntry) {
                                try { await applyMove(selectedEntry.id, pd.kind, pd.rowId, pd.day, pd.slot); }
                                catch (err) { flash("Ошибка: " + (err && err.message ? err.message : err)); }
                                selectedEntry = null;
                        }
                        return;
                }
                if (selectedEntry && selectedEntry.id !== pd.id) {
                        try { await applyMove(selectedEntry.id, pd.kind, pd.rowId, pd.day, pd.slot); }
                        catch (err) { flash("Ошибка: " + (err && err.message ? err.message : err)); }
                        selectedEntry = null;
                } else {
                        const isSame = selectedEntry && selectedEntry.id === pd.id;
                        selectedEntry = isSame ? null : { id: pd.id };
                        selectedClassId = isSame ? null : pd.rowId;
                }
        }
        function onPointerCancel() {
                try {
                        if (pendingDrag && pendingDrag.pointerId !== undefined) {
                                const el = document.elementFromPoint(lastPointer.x, lastPointer.y);
                                if (el && el.releasePointerCapture) el.releasePointerCapture(pendingDrag.pointerId);
                        }
                } catch (err) { /* ignore */ }
                pendingDrag = null; dragging = false; ghost = null; dropTarget = null; startPos = null;
        }
        function onGlobalKeydown(e) {
                if (e.key === "Escape") {
                        if (dragging || pendingDrag) { pendingDrag = null; dragging = false; ghost = null; dropTarget = null; }
                        else if (selectedEntry) selectedEntry = null;
                }
        }

        // Attach pointer listeners to window in the CAPTURE phase (a table
        // cell can otherwise consume pointerup in the embedded webview) and
        // clean up on destroy.
        function attachDragListeners() {
                document.addEventListener("pointermove", onPointerMove, true);
                document.addEventListener("pointerup", onPointerUp, true);
                document.addEventListener("pointercancel", onPointerCancel, true);
                window.addEventListener("keydown", onGlobalKeydown, true);
        }
        function detachDragListeners() {
                document.removeEventListener("pointermove", onPointerMove, true);
                document.removeEventListener("pointerup", onPointerUp, true);
                document.removeEventListener("pointercancel", onPointerCancel, true);
                window.removeEventListener("keydown", onGlobalKeydown, true);
        }
        onMount(() => { attachDragListeners(); detectPreciseSolver(); return () => detachDragListeners(); });
        function cellAt(kind, id, day, slot) {
                const arr = scheduleIndex.idx.get(cellKey(kind, id, day, slot));
                if (!arr || !arr.length) return null;
                const e = arr[0];
                return {
                        id: e.id,
                        subject_id: e.subject_id,
                        teacher_id: e.teacher_id,
                        room_id: e.room_id,
                        conflict: conflictIDs.has(e.id),
                        label: subjName(subjects, e.subject_id) + " (" + teachName(teachers, e.teacher_id) + ") " + (roomById.get(e.room_id)?.name || "")
                };
        }

// cellSubs: остальные уроки в той же ячейке (параллельные подгруппы,
// либо дубль-конфликт того же учителя/кабинета). Первая запись под
// ключом — сама ячейка (cellAt) и исключается; для класса ячейка
// родителя дополнительно читает ключи его подгрупп.
function cellSubs(kind, id, day, slot) {
        const { idx, pos } = scheduleIndex;
        const self = idx.get(cellKey(kind, id, day, slot)) || [];
        const excludeId = self.length ? self[0].id : -1;
        if (kind !== "class") {
                return self.filter((en) => en.id !== excludeId);
        }
        const merged = self.slice();
        for (const kid of (kidsOf.get(id) || [])) {
                const kidArr = idx.get(cellKey("class", kid, day, slot));
                if (kidArr) for (const en of kidArr) merged.push(en);
        }
        merged.sort((a, b) => pos.get(a.id) - pos.get(b.id));
        return merged.filter((en) => en.id !== excludeId);
}
        // Тело класса: сам класс + его подгруппы.
        function classBody(classId) {
                return [classId, ...(kidsOf.get(classId) || [])];
        }
        function subParentOf(classId) {
                return classById.get(classId)?.subgroup_of || null;
        }
        // Делёная пара (модель aSc): у урока подгруппы есть «сестринский»
        // урок ТОГО ЖЕ предмета у другой подгруппы того же родителя.
        // Половинки обязаны стоять параллельно и двигаться как группа.
        // Индекс строится реактивно — раньше dividedTwinLesson делал
        // два Array.find по урокам на каждый вызов (квадрат в подсчёте
        // конфликтов).
        let dividedTwinOf = new Map();
        $: {
                void classes; // зависимость: subParentOf читает classById
                const m = new Map();
                const byKey = new Map(); // `${parent}|${subject}` → первый урок семьи
                for (const l of lessons) {
                        const parent = subParentOf(l.class_id);
                        if (!parent) continue;
                        const key = parent + "|" + l.subject_id;
                        const first = byKey.get(key);
                        if (first) {
                                // пара ссылается друг на друга; при 3+ подгруппах
                                // первый остаётся со второй (как в исходном поиске)
                                m.set(l.id, first);
                                if (!m.has(first.id)) m.set(first.id, l);
                        } else {
                                byKey.set(key, l);
                        }
                }
                dividedTwinOf = m;
        }
        function dividedTwinLesson(lessonId) {
                return dividedTwinOf.get(lessonId) || null;
        }
        // Компактная подпись для обзора «вся школа»: краткое имя предмета
        // (short_name, если задан) — полный контекст в подсказке ячейки.
        function isSubRow(id) {
                const c = classById.get(id);
                return !!(c && c.subgroup_of);
        }
        // Имя родителя, если класс — подгруппа.
        function subParentName(id) {
                const c = classById.get(id);
                if (!c || !c.subgroup_of) return null;
                return classById.get(c.subgroup_of)?.name || null;
        }
        function subjShort(list, id) {
                const s = subjectById.get(id);
                if (!s) return "?";
                return s.short_name || s.name;
        }
        // Клик по заголовку мини-таблицы — открыть этот класс отдельно
        // (в подробном виде с drag&drop), на нужной странице пагинации.
        function focusRow(id) {
                viewMode = "class";
                const idx = classes.findIndex((c) => c.id === id);
                if (idx >= 0) classPage = Math.floor(idx / classesPerPage);
        }

        async function exportJSON() {
                const snap = await ExportAll(activeSchoolID);
                await saveFile("school.json", JSON.stringify(snap, null, 2), "application/json", false);
        }
        async function importJSON(ev) {
                const text = await ev.target.files[0].text();
                const restored = await ImportAll(text);
                // An imported backup can create a new school with a different
                // local ID. Show that restored school immediately.
                if (restored && restored.id) activeSchoolID = restored.id;
                await loadSchools();
                await reloadRefs();
                await reloadSchedule();
                flash("Импортировано");
        }
        async function exportCSV() {
                const csv = await ScheduleCSV(activeSchoolID, days, slots);
                await saveFile("schedule.csv", csv, "text/csv", false);
        }

        // exportPDF asks the Go backend to render the PDF (via internal/pdf +
        // signintech/gopdf + an embedded DejaVu Sans font for Cyrillic) and
        // returns it as base64. We pass it straight to the existing SaveExport
        // flow that writes to ~/Downloads.
        //
        // Old behavior (jsPDF) was unreliable inside the Wails WebKit webview
        // (datauristring output could yield a syntactically valid but empty
        // PDF) and the default fonts had no Cyrillic glyphs. Moving the
        // rendering to Go fixes both.
        async function exportPDF() {
                try {
                        const options = {
                                mode: exportMode,
                                page_size: pageSize,
                                orientation: orientation,
                                show_teacher: pdfShowTeacher,
                                show_room: pdfShowRoom,
                                weekdays_only: pdfWeekdaysOnly,
                                bw: pdfBW,
                                days: days,
                                slots: slots,
                                days_mask: schoolDaysMask,
                        };
                        const b64 = await ExportPDF(activeSchoolID, JSON.stringify(options));
                        if (!b64) throw new Error("PDF не содержит данных");
                        const defName = "Расписание_" + fileSlug(pdfSchoolName()) + "_" + new Date().toISOString().slice(0, 10) + ".pdf";
                        try {
                                await saveFile(defName, b64, "application/pdf", true);
                        } catch (err) {
                                flash("Ошибка сохранения PDF: " + (err && err.message ? err.message : err));
                        }
                } catch (e) {
                        flash("Ошибка генерации PDF: " + (e && e.message ? e.message : e));
                }
        }
        // Насыщенные цвета чипов — тот же список, что в app.go для PDF.
        // Белый текст читается на любом из этих цветов.
        function subjectColor(sid) {
                const palette = ["#3b82f6","#10b981","#f59e0b","#8b5cf6","#ec4899","#06b6d4","#6366f1","#14b8a6","#f97316","#a855f7","#0ea5e9","#ca8a04","#65a30d","#db2777"];
                return palette[Math.abs((sid || 0)) % palette.length];
        }
        function pdfSchoolName() {
                return (schools.find(s => s.id === activeSchoolID)?.name) || "Школа";
        }
        function fileSlug(s) {
                return String(s).replace(/[\\/:*?"<>|]/g, "_").replace(/\s+/g, "_");
        }

        let saveModal = null; // { filename, content, mime, isBase64 }
        async function saveFile(filename, content, mime, isBase64) {
                saveModal = { filename, content, mime, isBase64 };
        }
        async function confirmSave() {
                if (!saveModal) return;
                const name = saveModal.filename || "export";
                const b64 = saveModal.isBase64 ? saveModal.content : btoa(unescape(encodeURIComponent(saveModal.content)));
                try {
                        const path = await SaveExport(name, b64);
                        flash("Сохранено: " + path);
                } catch (e) {
                        flash("Ошибка сохранения: " + (e && e.message ? e.message : e));
                }
                saveModal = null;
        }
        function cancelSave() { saveModal = null; }
        // Справочники гоняются обычным JSON, не CSV: меньше зависимостей,
        // формат совпадает с тем, что и так ходит через API. В файл пишем
        // имена вместо id — файл переносим между школами и базами.
        async function downloadRefsJSON(entity) {
                let data;
                if (entity === "teachers") data = teachers.map((t) => ({ name: t.name, short_name: t.short_name || "" }));
                else if (entity === "subjects") data = subjects.map((s) => ({ name: s.name, short_name: s.short_name || "", requires_room_type: s.requires_room_type || "any" }));
                else if (entity === "rooms") data = rooms.map((r) => ({ name: r.name, room_type: r.room_type || "any" }));
                else if (entity === "classes") data = classes.map((c) => ({ name: c.name, room: c.room_id ? (roomName(rooms, c.room_id) || "") : "", subgroup_of: c.subgroup_of ? (className(classes, c.subgroup_of) || "") : "" }));
                else if (entity === "lessons") data = lessons.map((l) => ({ class: className(classes, l.class_id), subject: subjName(subjects, l.subject_id), teacher: teachName(teachers, l.teacher_id), hours_per_week: l.hours_per_week }));
                else if (entity === "periods") data = bellPeriods.slice(0, slots).map((p) => ({ start: p.start || "", end: p.end || "" }));
                else return;
                await saveFile(entity + ".json", JSON.stringify(data, null, 2), "application/json", false);
        }
        async function importRefsJSON(entity, e) {
                const file = e.target.files && e.target.files[0];
                if (!file) return;
                try {
                        const items = JSON.parse(await file.text());
                        if (!Array.isArray(items)) throw new Error("ожидался JSON-массив");
                        let n = 0;
                        for (const it of items) {
                                if (entity === "teachers") {
                                        if (!it.name) continue;
                                        await CreateTeacher({ school_id: activeSchoolID, name: String(it.name), short_name: String(it.short_name || "") });
                                } else if (entity === "subjects") {
                                        if (!it.name) continue;
                                        await CreateSubject({ school_id: activeSchoolID, name: String(it.name), short_name: String(it.short_name || ""), requires_room_type: String(it.requires_room_type || "any") });
                                } else if (entity === "rooms") {
                                        if (!it.name) continue;
                                        await CreateRoom({ school_id: activeSchoolID, name: String(it.name), room_type: String(it.room_type || "any") });
                                } else if (entity === "classes") {
                                        if (!it.name) continue;
                                        let room_id = 0;
                                        if (it.room) { const r = rooms.find((x) => x.name === it.room); if (r) room_id = r.id; }
                                        let subgroup_of = null;
                                        if (it.subgroup_of) { const p = classes.find((x) => x.name === it.subgroup_of); if (p) subgroup_of = p.id; }
                                        const created = await CreateClass({ school_id: activeSchoolID, name: String(it.name), room_id, subgroup_of });
                                        classes = [...classes, created]; // родитель должен быть виден следующим строкам файла
                                } else if (entity === "lessons") {
                                        const c = classes.find((x) => x.name === it.class);
                                        const s = subjects.find((x) => x.name === it.subject);
                                        const t = teachers.find((x) => x.name === it.teacher || (x.short_name && x.short_name === it.teacher));
                                        if (!c || !s || !t) throw new Error("не найдено в справочниках: " + JSON.stringify(it));
                                        await CreateLesson({ school_id: activeSchoolID, class_id: c.id, subject_id: s.id, teacher_id: t.id, hours_per_week: Number(it.hours_per_week) || 1 });
                                } else if (entity === "periods") {
                                        // массив звонков заменяет расписание звонков целиком
                                        const ps = items.map((p) => ({ start: String(p.start || ""), end: String(p.end || "") }));
                                        await UpdateSchoolSettings(activeSchoolID, JSON.stringify({ days, slots: ps.length, days_mask: schoolDaysMask, periods: ps }));
                                        await loadSettings();
                                        n = ps.length;
                                        break;
                                }
                                n++;
                        }
                        await reloadRefs();
                        flash("Импортировано: " + n);
                } catch (err) {
                        flash("Ошибка импорта: " + (err && err.message ? err.message : err));
                } finally {
                        e.target.value = "";
                }
        }

        // Резолверы имён — O(1) через map (list остаётся в сигнатуре как
        // синтаксическая зависимость реактивности, сам не используется).
        function className(list, id) { const x = classById.get(id); return x ? x.name : "?"; }
        function roomName(list, id) { const x = roomById.get(id); return x ? x.name : "?"; }
        function subjName(list, id) { const x = subjectById.get(id); return x ? x.name : "?"; }
        function teachName(list, id) { const x = teacherById.get(id); return x ? (x.short_name || x.name) : "?"; }
        function dayName(d) { return ["Пн","Вт","Ср","Чт","Пт","Сб","Вс"][d] || ("Д" + (d + 1)); }
        function sI(i) { return i + 1; }
        function periodLabel(si) { const p = bellPeriods[si]; return p && p.start ? p.start + "–" + p.end : ""; }
        function slotLabel(si) {
                const lbl = periodLabel(si);
                return "П" + (si + 1) + (lbl ? " " + lbl : "");
        }

        // -- Constraint UI helpers --
        const CONSTRAINT_FIELDS = {
                teacher_unavailable:    { day: true,  slots: true,  value: false, valueLabel: "" },
                class_unavailable:      { day: true,  slots: true,  value: false, valueLabel: "" },
                room_unavailable:       { day: true,  slots: true,  value: false, valueLabel: "" },
                max_consecutive:        { day: false, slots: false, value: true,  valueLabel: "Макс. подряд" },
                max_consecutive_subject:{ day: false, slots: false, value: true,  valueLabel: "не более подряд" },
                lunch_break:            { day: false, slots: true,  value: false, valueLabel: "" },
                max_lessons_per_day:    { day: false, slots: false, value: true,  valueLabel: "Макс. в день" },
                min_lessons_per_day:    { day: false, slots: false, value: true,  valueLabel: "Мин. в день" },
                prefer_morning:         { day: false, slots: true,  value: false, valueLabel: "" },
                max_gaps:               { day: false, slots: false, value: true,  valueLabel: "Макс. окон" },
        };
        const CONSTRAINT_LABELS = {
                teacher_unavailable:    "Учитель недоступен",
                class_unavailable:      "Класс недоступен",
                room_unavailable:       "Кабинет недоступен",
                max_consecutive:        "Макс. подряд уроков",
                max_consecutive_subject:"Один предмет — не более N подряд",
                lunch_break:            "Обеденный перерыв",
                max_lessons_per_day:    "Макс. уроков в день",
                min_lessons_per_day:    "Мин. уроков в день",
                prefer_morning:         "Желательно утро",
                max_gaps:               "Макс. окон",
        };
        function constraintTypeLabel(t) { return CONSTRAINT_LABELS[t] || t; }
        function constraintFields(t) { return CONSTRAINT_FIELDS[t] || { day: false, slots: false, value: false, valueLabel: "" }; }
        function constraintEntityLabel(c) {
                if (c.entity_type === "school") return "вся школа";
                if (c.entity_type === "subject") return subjName(subjects, c.entity_id);
                const list = c.entity_type === "teacher" ? teachers : c.entity_type === "class" ? classes : rooms;
                const item = list.find(x => x.id === c.entity_id);
                return item ? item.name : "#" + c.entity_id;
        }
        function constraintDetail(c) {
                const f = constraintFields(c.type);
                const parts = [];
                if (f.day && c.day_of_week != null) parts.push(dayName(c.day_of_week));
                if (f.slots) {
                        if (c.timeslot_start != null && c.timeslot_end != null) {
                                parts.push("П" + (c.timeslot_start + 1) + "–П" + (c.timeslot_end + 1));
                        } else if (c.timeslot_start != null) {
                                parts.push("с П" + (c.timeslot_start + 1));
                        } else if (c.timeslot_end != null) {
                                parts.push("по П" + (c.timeslot_end + 1));
                        }
                }
                if (f.value) parts.push(String(c.weight));
                return parts.join(", ");
        }
        function constraintSummary(c) {
                const detail = constraintDetail(c);
                return constraintTypeLabel(c.type) + " · " + constraintEntityLabel(c) + (detail ? " (" + detail + ")" : "") + " · " + (c.is_hard ? "🔒 жёсткое" : "📝 мягкое");
        }

        // -- DnD helper for compact labels in flash --
        function cellLabelShort(e) {
                if (!e) return "?";
                return subjName(subjects, e.subject_id) + " (" + teachName(teachers, e.teacher_id) + ")";
        }

        $: scheduleGridVM = {
                overviewMode, grid, slots, isSubRow, subParentName, focusRow, periodLabel,
                activeDayIdx, dayName, selectedEntry, dropTarget, kind, onPointerDown,
                cellSubs, subjectColor, subjShort, subjName, teachName, pdfShowTeacher,
                pdfShowRoom, roomById, conflictIDs, classById, classes, compact, ghost,
                removeEntry, cellAt, subjects, teachers
        };
        $: placementVM = { report, lessons };
        $: issuesVM = {
                report, subjects, classes, teachers, subjName, className, teachName,
                dayName, generate, onGoSchedule: () => tab = "schedule"
        };

        loadSchools();
</script>

<div class="app">
        <AppSidebar activeTab={tab} onChange={(nextTab) => tab = nextTab} />

        <div class="content">
                <header class="topbar">
                        <div class="school">
                                <select bind:value={activeSchoolID} on:change={async () => { await reloadRefs(); await loadSettings(); }}>
                                        {#each schools as sc}<option value={sc.id}>{sc.name}</option>{/each}
                                </select>
                                {#if schools.length === 0}<span class="muted">нет школ</span>{/if}
                                <input class="school-new" bind:value={newSchoolName} placeholder="Новая школа" />
                                <button class="primary sm" on:click={createSchool}>+ Школа</button>
                                {#if activeSchoolID}
                                <button class="danger sm" on:click={deleteSchool} title="Удалить текущую школу со всеми данными">🗑</button>
                                {/if}
                        </div>
                        {#if msg}<div class="toast">{msg}</div>{/if}
                        <span class="ver">v{APP_VERSION}</span>
                </header>

                <main>
                        {#if tab === "refs"}
                                <ReferencesTab
                                        bind:teachers bind:subjects bind:classes bind:rooms
                                        bind:t bind:s bind:c bind:r bind:editing
                                        addTeacher={addTeacher} addSubject={addSubject} addClass={addClass} addRoom={addRoom}
                                        saveEdit={saveEdit} cancelEdit={cancelEdit} startEdit={startEdit}
                                        removeTeacher={removeTeacher} removeSubject={removeSubject} removeClass={removeClass} removeRoom={removeRoom}
                                        downloadRefsJSON={downloadRefsJSON} importRefsJSON={importRefsJSON}
                                />
                        {:else if tab === "lessons"}
                                <LessonsTab
                                        classes={classes} bind:lessons bind:curClass bind:l
                                        subjects={subjects} teachers={teachers} subjName={subjName}
                                        updateLesson={updateLesson} removeLesson={removeLesson} addLessonForClass={addLessonForClass}
                                        downloadRefsJSON={downloadRefsJSON} importRefsJSON={importRefsJSON}
                                />
                        {:else if tab === "constraints"}
                                <ConstraintsTab
                                        DAY_NAMES={DAY_NAMES} activeDayIdx={activeDayIdx} schoolDaysMask={schoolDaysMask}
                                        toggleSchoolDay={toggleSchoolDay} consPickKind={consPickKind} setPickKind={setPickKind}
                                        bind:consPickId consPickList={consPickList} consPickEntity={consPickEntity} pickName={pickName}
                                        slots={slots} periodLabel={periodLabel} dayName={dayName}
                                        isForbidden={isForbidden} toggleForbidden={toggleForbidden}
                                        constraints={constraints} constraintSummary={constraintSummary} removeConstraint={removeConstraint}
                                        subjects={subjects} bind:ruleSubject bind:ruleMax addRunRule={addRunRule}
                                />
                        {:else if tab === "settings"}
                                <SettingsTab
                                        bind:days bind:slots onSlotsChange={onSlotsChange} saveSettings={saveSettings}
                                        downloadRefsJSON={downloadRefsJSON} importRefsJSON={importRefsJSON}
                                        bind:bellPeriods normalizeTime={normalizeTime}
                                />
                        {:else if tab === "schedule"}
                                <section class="card">
                                        <div class="gen-bar">
                                                <span class="grid-info">{days} дней · {slots} уроков в день</span>
                                                <button class="primary" on:click={generate} disabled={isGenerating}>{isGenerating ? "⏳ Генерирую..." : "⚙ Сгенерировать"}</button>
                                                <button on:click={undo} disabled={isGenerating}>↶ Отменить</button>
                                                <label class="chk" class:warn={!hasPreciseSolver} title={hasPreciseSolver ? "OR-Tools CP-SAT доступен в этой сборке" : "OR-Tools CP-SAT НЕ скомпилирован в эту сборку — будет использован быстрый эвристический решатель"}><input type="checkbox" bind:checked={usePrecise} disabled={isGenerating} /> точный CP-SAT (OR-Tools){#if !hasPreciseSolver} <span class="badge-warn">недоступно</span>{/if}</label>
                                                <label class="chk" disabled={isGenerating || !usePrecise}>
                                                        таймаут CP-SAT:
                                                        <select bind:value={solveTimeoutPreset} disabled={isGenerating || !usePrecise}>
                                                                <option value="none">без лимита</option>
                                                                <option value="30">30 сек</option>
                                                                <option value="90">90 сек</option>
                                                                <option value="custom">другое</option>
                                                        </select>
                                                        {#if solveTimeoutPreset === "custom"}
                                                                <input type="number" bind:value={solveTimeoutCustom} min="0" style="width:5ch" disabled={isGenerating || !usePrecise} />
                                                        {/if}
                                                </label>
<span class="sep"></span>
<label>Вид:
                                                        <select bind:value={viewMode} disabled={isGenerating}>
                                                                <option value="school">вся школа (обзор)</option>
                                                                <option value="class">по классам</option>
                                                                <option value="teacher">по учителям</option>
                                                                <option value="room">по кабинетам</option>
                                                        </select>
                                                </label>
                                                {#if !overviewMode}
                                                <label class="chk"><input type="checkbox" bind:checked={compact} disabled={isGenerating} /> компактный</label>
                                                {/if}
                                                {#if !overviewMode && kind === "class" && rows.length > classesPerPage}
                                                        <span class="pager">
                                                                <button class="sm" on:click={() => classPage = Math.max(0, classPage - 1)} disabled={classPage === 0 || isGenerating}>‹</button>
                                                                <span>{classPage + 1}/{totalClassPages}</span>
                                                                <button class="sm" on:click={() => classPage = Math.min(totalClassPages - 1, classPage + 1)} disabled={classPage >= totalClassPages - 1 || isGenerating}>›</button>
                                                        </span>
                                                {/if}
                                                <span class="sep"></span>
                                                {#if isGenerating && genResult}
                                                <div class="gen-progress">
                                                        <span class="prog-label">Генерация: {genResult.placed} из {genResult.total} ч</span>
                                                        <div class="prog-bar"><div class="prog-fill" style="width:{genResult.total ? Math.round(genResult.placed / genResult.total * 100) : 0}%"></div></div>
                                                        <span class="prog-pct">{genResult.total ? Math.round(genResult.placed / genResult.total * 100) : 0}%</span>
                                                </div>
                                                {/if}
                                                <div class="export-panel">
                                                <span class="panel-label">ЭКСПОРТ</span>
                                                <label>Стр.:
                                                        <select bind:value={pageSize} disabled={isGenerating}>
                                                                <option value="A0">A0</option><option value="A1">A1</option><option value="A2">A2</option><option value="A3">A3</option><option value="A4">A4</option>
                                                        </select>
                                                </label>
                                                <label>Ориент.:
                                                        <select bind:value={orientation} disabled={isGenerating}>
                                                                <option value="landscape">альбомн.</option><option value="portrait">книжн.</option>
                                                        </select>
                                                </label>
                                                <label class="chk"><input type="checkbox" bind:checked={pdfShowTeacher} disabled={isGenerating} /> учителя</label>
                                                <label class="chk"><input type="checkbox" bind:checked={pdfShowRoom} disabled={isGenerating} /> кабинеты</label>
                                                <label class="chk"><input type="checkbox" bind:checked={pdfWeekdaysOnly} disabled={isGenerating} /> будни</label>
                                                <label class="chk"><input type="checkbox" bind:checked={pdfBW} disabled={isGenerating} /> ч/б</label>
                                                <button on:click={exportPDF} disabled={isGenerating}>⬇ PDF</button>
                                                <button on:click={exportCSV} disabled={isGenerating}>⬇ CSV</button>
                                                <button on:click={exportJSON} disabled={isGenerating}>⬇ JSON</button>
                                                <label class="file">⬆ JSON<input type="file" accept="application/json" on:change={importJSON} disabled={isGenerating} /></label>
                                                </div>
                                        </div>
                                        {#if genResult}<p class="status">Размещено <b>{genResult.placed}/{genResult.total}</b> · мягких нарушений: <b>{genResult.violations}</b></p>{/if}
                                        <p class="hint">Чтобы переместить урок: зажмите и перетащите ячейку в другую (drag) либо кликните урок, затем кликните целевую ячейку. Во время перетаскивания целевая ячейка подсвечивается. Повторный клик по выделенному (или Esc) снимает выделение. ✕ в ячейке — удалить.</p>
                                        {#if schedule.length === 0}
                                                <p class="empty">Расписание пусто. Добавьте уроки и нажмите «Сгенерировать».</p>
                                        {:else}
                                                <ScheduleGrid vm={scheduleGridVM} />
                                                {#if report.conflicts.length || report.violations.length}
                                                        <div class="report">
                                                                <h3>Отчёт по конфликтам</h3>
                                                                {#if report.conflicts.length}
                                                                        <div class="rep-sec"><b>Накладки ({report.conflicts.length}):</b>
                                                                                <ul>{#each report.conflicts as c}<li>{dayName(c.day)} П{c.slot + 1}: {c.type} — {#each c.items as it, i}{it.subject} ({it.who}){#if i < c.items.length - 1}, {/if}{/each}</li>{/each}</ul>
                                                                        </div>
                                                                {/if}
                                                                {#if report.violations.length}
                                                                        <div class="rep-sec"><b>Нарушено жёсткое ограничение ({report.violations.length}):</b>
                                                                                <ul>{#each report.violations as v}<li>{dayName(v.day)} П{v.slot + 1}: {v.what} — урок стоит в запрещённой ячейке</li>{/each}</ul>
                                                                        </div>
                                                                {/if}
                                                        </div>
                                                {/if}
                                        {/if}
                                        {#if lessons.length || schedule.length}
                                                <PlacementSummary report={placementVM.report} lessons={placementVM.lessons} />
                                        {/if}
                                </section>
                        {:else if tab === "issues"}
                                <IssuesTab vm={issuesVM} />
                        {/if}
                </main>

                {#if saveModal}
                        <div class="modal-backdrop" role="button" tabindex="-1" on:click={cancelSave} on:keydown={(e) => { if (e.key === 'Escape') cancelSave(); }}>
                                <div class="modal" role="dialog" tabindex="0" on:click|stopPropagation on:keydown|stopPropagation>
                                        <h3>Сохранить файл</h3>
                                        <p class="muted">Файл будет записан в папку ~/Downloads</p>
                                        <input class="modal-input" bind:value={saveModal.filename} placeholder="имя файла" on:keydown={(e) => { if (e.key === "Enter") confirmSave(); }} />
                                        <div class="modal-actions">
                                                <button on:click={cancelSave}>Отмена</button>
                                                <button class="primary" on:click={confirmSave}>Сохранить</button>
                                        </div>
                                </div>
                        </div>
                {/if}

                {#if confirmBox}
                        <div class="modal-backdrop" role="button" tabindex="-1" on:click={() => settleConfirm(false)} on:keydown={(e) => { if (e.key === 'Escape') settleConfirm(false); }}>
                                <div class="modal" role="dialog" tabindex="0" on:click|stopPropagation on:keydown|stopPropagation>
                                        <h3>Подтверждение</h3>
                                        <p class="modal-msg">{confirmBox.message}</p>
                                        <div class="modal-actions">
                                                <button on:click={() => settleConfirm(false)}>Отмена</button>
                                                <button class="danger" on:click={() => settleConfirm(true)}>{confirmBox.confirmLabel}</button>
                                        </div>
                                </div>
                        </div>
                {/if}


        </div>
</div>

<style>
        :global(body) { margin: 0; }
        .app {
                display: flex;
                min-height: 100vh;
                font-family: ui-sans-serif, system-ui, -apple-system, "Segoe UI", Roboto, Arial, sans-serif;
                color: #1e293b;
                background: #f1f5f9;
        }
        .content { flex: 1; display: flex; flex-direction: column; min-width: 0; }
        .topbar {
                display: flex; align-items: center; gap: 14px;
                padding: 14px 24px; background: #ffffff; border-bottom: 1px solid #e2e8f0;
                position: sticky; top: 0; z-index: 5;
        }
        .school { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
        .school select { min-width: 160px; }
        .school-new { width: 150px; }
        main { padding: 24px; }

        input, select {
                background: #fff; color: #1e293b; border: 1px solid #cbd5e1;
                border-radius: 8px; padding: 7px 10px; font-size: 13px; outline: none;
        }
        input:focus, select:focus { border-color: #4f46e5; box-shadow: 0 0 0 3px #e0e7ff; }
        button {
                background: #e2e8f0; color: #1e293b; border: none; border-radius: 8px;
                padding: 8px 14px; cursor: pointer; font-size: 13px; font-weight: 500;
        }
        button:hover { background: #cbd5e1; }
        button.primary { background: #4f46e5; color: #fff; }
        button.primary:hover { background: #4338ca; }
        button.sm { padding: 5px 10px; font-size: 12px; }
        button.danger { background: #fee2e2; color: #b91c1c; }
        button.danger:hover { background: #fecaca; }
        .chk { display: inline-flex; align-items: center; gap: 5px; font-size: 13px; color: #475569; }
        .export-panel { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; padding: 6px 10px; background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 10px; }
        .export-panel .panel-label { font-size: 10px; font-weight: 700; letter-spacing: 0.5px; color: #64748b; text-transform: uppercase; margin-right: 2px; }
        .grid-info { font-size: 13px; font-weight: 600; color: #475569; background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 8px; padding: 5px 10px; }
        .chk.warn { color: #b45309; }
        .badge-warn { display: inline-block; background: #fde68a; color: #92400e; padding: 1px 6px; border-radius: 6px; font-size: 10px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.5px; }
        .chk input { width: auto; }
        .card {
                background: #fff; border: 1px solid #e2e8f0; border-radius: 14px;
                padding: 18px; box-shadow: 0 1px 2px rgba(15,23,42,0.04);
        }
        h3 { font-size: 14px; color: #334155; margin: 16px 0 8px; }

        .muted { color: #64748b; }

        .gen-bar { display: flex; gap: 10px; align-items: center; flex-wrap: wrap; margin-bottom: 12px; }
        .gen-bar .sep { flex-basis: 100%; height: 0; }
        .pager { display: inline-flex; align-items: center; gap: 6px; }
        .status { background: #ecfdf5; border: 1px solid #a7f3d0; color: #065f46; padding: 8px 12px; border-radius: 10px; font-size: 13px; }
        .hint { color: #64748b; font-size: 12px; margin: 6px 0 14px; }
        .empty { color: #94a3b8; padding: 24px; text-align: center; background: #f8fafc; border-radius: 10px; }
        .toast { background: #16a34a; color: #fff; padding: 8px 14px; border-radius: 10px; font-size: 13px; margin-left: auto; }
        .ver { margin-left: auto; font-size: 12px; color: #94a3b8; font-family: ui-monospace, monospace; }
        .modal-backdrop { position: fixed; inset: 0; background: rgba(15,23,42,0.5); display: flex; align-items: center; justify-content: center; z-index: 10000; }
        .modal { background: #fff; border-radius: 14px; padding: 22px; width: 360px; max-width: 90vw; box-shadow: 0 20px 60px rgba(0,0,0,0.3); }
        .modal h3 { margin: 0 0 6px; font-size: 16px; color: #0f172a; }
        .modal-input { width: 100%; margin: 12px 0; box-sizing: border-box; }
        .modal-msg { margin: 12px 0 0; font-size: 13.5px; line-height: 1.45; color: #334155; }
        .modal-actions { display: flex; justify-content: flex-end; gap: 8px; }
        .report { margin-top: 14px; border: 1px solid #fecaca; background: #fef2f2; border-radius: 10px; padding: 10px 14px; font-size: 13px; }
        .report h3 { margin: 0 0 6px; font-size: 13px; color: #b91c1c; }
        .rep-sec { margin: 6px 0; }
        .rep-sec ul { margin: 4px 0 0; padding-left: 18px; }
        .rep-sec li { margin: 2px 0; }

        /* Прогресс-бар генерации */
        .gen-progress { display: flex; align-items: center; gap: 10px; margin-top: 8px; }
        .prog-label { font-size: 13px; color: #64748b; min-width: 150px; }
        .prog-bar { flex: 1; height: 10px; background: #e2e8f0; border-radius: 5px; overflow: hidden; }
        .prog-fill { height: 100%; background: #4f46e5; transition: width 0.3s ease; }
        .prog-pct { font-size: 13px; font-weight: 700; color: #4f46e5; min-width: 35px; }

        @media (max-width: 980px) {
                .topbar { align-items: flex-start; flex-wrap: wrap; }
                .school { flex: 1 1 100%; }
        }
        @media (max-width: 720px) {
                .app { flex-direction: column; }
                .topbar { position: static; padding: 12px 16px; }
                main { padding: 16px; }
                .export-panel { width: 100%; box-sizing: border-box; }
        }
        @media (max-width: 480px) {
                .topbar, main { padding-left: 12px; padding-right: 12px; }
                .card { padding: 14px; }
        }
</style>
