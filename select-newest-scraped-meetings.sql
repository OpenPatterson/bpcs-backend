SELECT m.*
FROM
    (SELECT meetingID, max(id) as id
    FROM meetings
    GROUP BY meetingID
    ) AS mx 
JOIN meetings m ON m.meetingID = mx.meetingID AND mx.id = m.id
ORDER BY meetingID;